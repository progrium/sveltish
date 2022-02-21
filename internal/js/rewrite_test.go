package js

import (
	"bytes"
	"testing"
)

func TestLexRewriteAssignment(t *testing.T) {
	testRewriteValue := []byte("REWRITTEN")
	testData := []struct {
		name    string
		input   []byte
		targets [][]byte
		output  []byte
	}{
		{
			"NoAssignments",
			[]byte("func({ some: 'test' });"),
			[][]byte{},
			[]byte("func({ some: 'test' });"),
		},
		{
			"DotAssignment",
			[]byte("value.test = 'test';"),
			[][]byte{},
			[]byte("value.test = 'test';"),
		},
		{
			"JustAnAssignment",
			[]byte("value = 'some test';"),
			[][]byte{
				[]byte("value = 'some test'"),
			},
			[]byte("REWRITTEN;"),
		},
		{
			"JustPlusAssignment",
			[]byte("value =+ 'some test';"),
			[][]byte{
				[]byte("value =+ 'some test'"),
			},
			[]byte("REWRITTEN;"),
		},
		{
			"MultipleAssignments",
			[]byte(`
				value = 'some test';
				another = 'value';
			`),
			[][]byte{
				[]byte("value = 'some test'"),
				[]byte("another = 'value'"),
			},
			[]byte(`
				REWRITTEN;
				REWRITTEN;
			`),
		},
		{
			"AssignmentsAndNonAssignments",
			[]byte(`
				// Some comment
				value = 'some test';

				if (value === 'some test') {
					another = 'value';
				}
			`),
			[][]byte{
				[]byte("value = 'some test'"),
				[]byte("another = 'value'"),
			},
			[]byte(`
				// Some comment
				REWRITTEN;

				if (value === 'some test') {
					REWRITTEN;
				}
			`),
		},
	}

	for _, td := range testData {
		td := td
		t.Run(td.name, func(t *testing.T) {
			foundTargets := [][]byte{}
			result := RewriteAssignments(td.input, func(data []byte) []byte {
				foundTargets = append(foundTargets, data)

				return testRewriteValue
			})

			if len(td.targets) != len(foundTargets) {
				t.Fatalf("Expected to find %d targets but got %d", len(td.targets), len(foundTargets))
			}
			for i, tg := range td.targets {
				if bytes.Compare(tg, foundTargets[i]) != 0 {
					t.Fatalf("Expected data %q but got %q", tg, foundTargets[i])
				}
			}

			if bytes.Compare(td.output, result) != 0 {
				t.Fatalf("Expected result to be %q but got %q", td.output, result)
			}
		})
	}
}

func TestLexRewriteVarNames(t *testing.T) {
	testRewriteValue := []byte("REWRITTEN")
	testData := []struct {
		name    string
		input   []byte
		targets [][]byte
		output  []byte
	}{
		{
			"NoVarNames",
			[]byte("1 + 1 == 2;"),
			[][]byte{},
			[]byte("1 + 1 == 2;"),
		},
		{
			"SingleVarName",
			[]byte("test = 'value';"),
			[][]byte{
				[]byte("test"),
			},
			[]byte("REWRITTEN = 'value';"),
		},
		{
			"MultipleVarNames",
			[]byte("test = value.method();"),
			[][]byte{
				[]byte("test"),
				[]byte("value"),
			},
			[]byte("REWRITTEN = REWRITTEN.method();"),
		},
	}

	for _, td := range testData {
		td := td
		t.Run(td.name, func(t *testing.T) {
			foundTargets := [][]byte{}
			result := RewriteVarNames(td.input, func(data []byte) []byte {
				foundTargets = append(foundTargets, data)

				return testRewriteValue
			})

			if len(td.targets) != len(foundTargets) {
				t.Fatalf("Expected to find %d targets but got %d", len(td.targets), len(foundTargets))
			}
			for i, tg := range td.targets {
				if bytes.Compare(tg, foundTargets[i]) != 0 {
					t.Fatalf("Expected data %q but got %q", tg, foundTargets[i])
				}
			}

			if bytes.Compare(td.output, result) != 0 {
				t.Fatalf("Expected result to be %q but got %q", td.output, result)
			}
		})
	}
}
