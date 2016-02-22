package setup

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/mholt/caddy/middleware"
	"github.com/mholt/caddy/middleware/mmark"
)

func TestMMark(t *testing.T) {

	c := NewTestController(`mmark /blog`)

	mid, err := MMark(c)

	if err != nil {
		t.Errorf("Expected no errors, got: %v", err)
	}

	if mid == nil {
		t.Fatal("Expected middleware, was nil instead")
	}

	handler := mid(EmptyNext)
	myHandler, ok := handler.(mmark.MMark)

	if !ok {
		t.Fatalf("Expected handler to be type MMark, got: %#v", handler)
	}

	if myHandler.Configs[0].PathScope != "/blog" {
		t.Errorf("Expected /blog as the Path Scope")
	}
	if fmt.Sprint(myHandler.Configs[0].Extensions) != fmt.Sprint([]string{".md", ".markdown", ".mmark"}) {
		t.Errorf("Expected .md, .markdown, and .mmark as default extensions")
	}
}

func TestMMarkStaticGen(t *testing.T) {
	c := NewTestController(`mmark /blog {
	ext .md
	template tpl_with_include.html
	sitegen
}`)

	c.Root = "./testdata"
	mid, err := MMark(c)

	if err != nil {
		t.Errorf("Expected no errors, got: %v", err)
	}

	if mid == nil {
		t.Fatal("Expected middleware, was nil instead")
	}

	for _, start := range c.Startup {
		err := start()
		if err != nil {
			t.Errorf("Startup error: %v", err)
		}
	}

	next := middleware.HandlerFunc(func(w http.ResponseWriter, r *http.Request) (int, error) {
		t.Fatalf("Next shouldn't be called")
		return 0, nil
	})
	hndlr := mid(next)
	mkdwn, ok := hndlr.(mmark.MMark)
	if !ok {
		t.Fatalf("Was expecting a mmark.MMark but got %T", hndlr)
	}

	expectedStaticFiles := map[string]string{"/blog/first_post.md": "testdata/generated_site/blog/first_post.md/index.html"}
	if fmt.Sprint(expectedStaticFiles) != fmt.Sprint(mkdwn.Configs[0].StaticFiles) {
		t.Fatalf("Test expected StaticFiles to be  %s, but got %s",
			fmt.Sprint(expectedStaticFiles), fmt.Sprint(mkdwn.Configs[0].StaticFiles))
	}

	filePath := "testdata/generated_site/blog/first_post.md/index.html"
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("An error occured when getting the file information: %v", err)
	}

	html, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("An error occured when getting the file content: %v", err)
	}

	expectedBody := []byte(`<!DOCTYPE html>
<html>
<head>
<title>first_post</title>
</head>
<body>
<h1>Header title</h1>

<h1>Test h1</h1>

</body>
</html>
`)

	if !bytes.Equal(html, expectedBody) {
		t.Fatalf("Expected file content: %s got: %s", string(expectedBody), string(html))
	}

	fp := filepath.Join(c.Root, mmark.DefaultStaticDir)
	if err = os.RemoveAll(fp); err != nil {
		t.Errorf("Error while removing the generated static files: %v", err)
	}
}

func TestMMarkParse(t *testing.T) {
	tests := []struct {
		inputMMarkConfig    string
		shouldErr           bool
		expectedMMarkConfig []mmark.Config
	}{

		{`mmark /blog {
	ext .md .txt
	css /resources/css/blog.css
	js  /resources/js/blog.js
}`, false, []mmark.Config{{
			PathScope:  "/blog",
			Extensions: []string{".md", ".txt"},
			Styles:     []string{"/resources/css/blog.css"},
			Scripts:    []string{"/resources/js/blog.js"},
		}}},
		{`mmark /blog {
	ext .md
	template tpl_with_include.html
	sitegen
}`, false, []mmark.Config{{
			PathScope:  "/blog",
			Extensions: []string{".md"},
			Templates:  map[string]string{mmark.DefaultTemplate: "testdata/tpl_with_include.html"},
			StaticDir:  mmark.DefaultStaticDir,
		}}},
	}
	for i, test := range tests {
		c := NewTestController(test.inputMMarkConfig)
		c.Root = "./testdata"
		actualMMarkConfigs, err := mmarkParse(c)

		if err == nil && test.shouldErr {
			t.Errorf("Test %d didn't error, but it should have", i)
		} else if err != nil && !test.shouldErr {
			t.Errorf("Test %d errored, but it shouldn't have; got '%v'", i, err)
		}
		if len(actualMMarkConfigs) != len(test.expectedMMarkConfig) {
			t.Fatalf("Test %d expected %d no of WebSocket configs, but got %d ",
				i, len(test.expectedMMarkConfig), len(actualMMarkConfigs))
		}
		for j, actualMMarkConfig := range actualMMarkConfigs {

			if actualMMarkConfig.PathScope != test.expectedMMarkConfig[j].PathScope {
				t.Errorf("Test %d expected %dth MMark PathScope to be  %s  , but got %s",
					i, j, test.expectedMMarkConfig[j].PathScope, actualMMarkConfig.PathScope)
			}

			if fmt.Sprint(actualMMarkConfig.Styles) != fmt.Sprint(test.expectedMMarkConfig[j].Styles) {
				t.Errorf("Test %d expected %dth MMark Config Styles to be  %s  , but got %s",
					i, j, fmt.Sprint(test.expectedMMarkConfig[j].Styles), fmt.Sprint(actualMMarkConfig.Styles))
			}
			if fmt.Sprint(actualMMarkConfig.Scripts) != fmt.Sprint(test.expectedMMarkConfig[j].Scripts) {
				t.Errorf("Test %d expected %dth MMark Config Scripts to be  %s  , but got %s",
					i, j, fmt.Sprint(test.expectedMMarkConfig[j].Scripts), fmt.Sprint(actualMMarkConfig.Scripts))
			}
			if fmt.Sprint(actualMMarkConfig.Templates) != fmt.Sprint(test.expectedMMarkConfig[j].Templates) {
				t.Errorf("Test %d expected %dth MMark Config Templates to be  %s  , but got %s",
					i, j, fmt.Sprint(test.expectedMMarkConfig[j].Templates), fmt.Sprint(actualMMarkConfig.Templates))
			}
		}
	}

}
