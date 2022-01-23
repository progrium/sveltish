package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/progrium/sveltish"
)

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				logger.Println(r.Method, r.URL.Path, r.RemoteAddr)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

var elemPlugin = api.Plugin{
	Name: "elem",
	Setup: func(build api.PluginBuild) {
		build.OnLoad(api.OnLoadOptions{Filter: `\.elem$`},
			func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				bytes, err := sveltish.Build(args.Path)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				contents := string(bytes)
				return api.OnLoadResult{Contents: &contents}, nil
			})
	},
}

func main() {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("building...")

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{"example/src/main.js"},
		Bundle:      true,
		Outfile:     "example/public/build/main.js",
		Plugins:     []api.Plugin{elemPlugin},
		Write:       true,
	})

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Println(err.PluginName, err.Text, "Line:", err.Location.Line)
		}
		os.Exit(1)
	}

	logger.Println("listening on 8080...")

	log.Fatal(http.ListenAndServe(":8000",
		logging(logger)(
			http.FileServer(http.Dir("./example/public"))),
	))
}
