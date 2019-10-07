/*

Command genblog generates a static blog.

Add an article:

  genblog add <article-url-slug>

Serve site on localhost:

  genblog serve

Build site (HTML, images, Netlify files) to `public/`:

  genblog build

*/
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/russross/blackfriday"
)

var wd string

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	wd, err = os.Getwd()
	check(err)

	switch os.Args[1] {
	case "add":
		if len(os.Args) != 3 {
			usage()
		}
		id := os.Args[2]
		add(id)
		fmt.Println("genblog: Added article at ./articles/" + id + ".md")
	case "serve":
		fmt.Println("genblog: Serving blog at http://localhost:2000")
		serve(":2000")
	case "build":
		build()
		fmt.Println("genblog: Built blog at ./public")
	default:
		usage()
	}
}

func usage() {
	const s = `usage:
  genblog add <article-url-slug>
  genblog serve
  genblog build
`
	fmt.Fprint(os.Stderr, s)
	os.Exit(2)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Blog contains data loaded from config.json
type Blog struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Article contains data loaded from config.json and parsed Markdown files
type Article struct {
	Authors       []string      `json:"authors"`
	Body          template.HTML `json:"-"`
	Canonical     string        `json:"canonical,omitempty"`
	ID            string        `json:"id"`
	LastUpdated   string        `json:"-"`
	LastUpdatedIn string        `json:"-"`
	LastUpdatedOn string        `json:"-"`
	Published     string        `json:"published"`
	Tags          []string      `json:"tags,omitempty"`
	Title         string        `json:"-"`
	Updated       string        `json:"updated,omitempty"`
}

func add(id string) {
	blog, articles, _ := load()

	noDashes := strings.Replace(id, "-", " ", -1)
	noUnderscores := strings.Replace(noDashes, "_", " ", -1)
	title := strings.Title(noUnderscores)
	content := []byte("# " + title + "\n\n\n")
	check(ioutil.WriteFile(wd+"/articles/"+id+".md", content, 0644))

	u, err := user.Current()
	check(err)

	a := Article{
		Authors:   []string{u.Name},
		ID:        id,
		Published: time.Now().Format("2006-01-02"),
	}

	data := struct {
		Blog     Blog      `json:"blog"`
		Articles []Article `json:"articles"`
	}{
		Blog:     blog,
		Articles: append([]Article{a}, articles...),
	}
	config, err := json.MarshalIndent(data, "", "  ")
	check(err)
	check(ioutil.WriteFile(wd+"/config.json", config, 0644))
}

func serve(addr string) {
	http.HandleFunc("/", handler)
	check(http.ListenAndServe(addr, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	build()

	fmt.Println("genblog: " + r.Method + " " + r.URL.Path)

	// convert URLs like /intro to /intro.html for http.FileServer
	if r.URL.Path != "/" && path.Ext(r.URL.Path) == "" {
		r.URL.Path = r.URL.Path + ".html"
	}

	// use same headers specified in production, if present
	for k, v := range headers() {
		w.Header().Set(k, v)
	}

	fs := http.FileServer(http.Dir(wd + "/public"))
	fs.ServeHTTP(w, r)
}

func headers() map[string]string {
	result := make(map[string]string)

	dat, err := ioutil.ReadFile(wd + "/netlify/_headers")
	if err != nil {
		return result
	}

	for i, line := range strings.Split(string(dat), "\n") {
		if i == 0 { // ignore first line ("/*\n")
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		result[k] = v
	}

	return result
}

func build() {
	blog, articles, tags := load()

	if len(articles) == 0 {
		panic("error: no articles")
	}

	check(os.RemoveAll(wd + "/public"))
	check(os.MkdirAll(wd+"/public/images", os.ModePerm))

	indexPage := template.Must(template.ParseFiles(wd + "/ui/index.html"))
	f, err := os.Create("public/index.html")
	check(err)
	indexData := struct {
		Blog     Blog
		Articles []Article
		Tags     []string
	}{
		Blog:     blog,
		Articles: articles,
		Tags:     tags,
	}
	check(indexPage.Execute(f, indexData))

	articlePage := template.Must(template.ParseFiles(wd + "/ui/article.html"))
	for _, a := range articles {
		f, err := os.Create("public/" + a.ID + ".html")
		check(err)
		articleData := struct {
			Blog    Blog
			Article Article
		}{
			Blog:    blog,
			Article: a,
		}
		check(articlePage.Execute(f, articleData))
	}

	cmd := exec.Command("cp", "-a", wd+"/articles/images/.", wd+"/public/images")
	cmd.Run()

	cmd = exec.Command("cp", "-a", wd+"/netlify/.", wd+"/public")
	cmd.Run()
}

func load() (Blog, []Article, []string) {
	config, err := ioutil.ReadFile(wd + "/config.json")
	check(err)
	var data struct {
		Blog     Blog      `json:"blog"`
		Articles []Article `json:"articles"`
	}
	check(json.Unmarshal(config, &data))

	articles := make([]Article, len(data.Articles))
	tagMap := make(map[string]bool)

	for i, a := range data.Articles {
		title, body := preProcess("articles/" + a.ID + ".md")
		markdown := blackfriday.Run([]byte(body))

		lastUpdated := a.Published
		if a.Updated != "" {
			lastUpdated = a.Updated
		}
		t, err := time.Parse("2006-01-02", lastUpdated)
		check(err)
		lastUpdatedIn := t.Format("2006 January")
		lastUpdatedOn := t.Format("January 2, 2006")

		articles[i] = Article{
			Authors:       a.Authors,
			Body:          template.HTML(markdown),
			Canonical:     a.Canonical,
			ID:            a.ID,
			LastUpdated:   lastUpdated,
			LastUpdatedIn: lastUpdatedIn,
			LastUpdatedOn: lastUpdatedOn,
			Published:     a.Published,
			Tags:          a.Tags,
			Title:         title,
			Updated:       a.Updated,
		}
		for _, t := range a.Tags {
			tagMap[t] = true
		}
	}

	tags := []string{}
	for tag := range tagMap {
		tags = append(tags, tag)
	}

	return data.Blog, articles, tags
}

/*
preProcess scans the Markdown document at filepath line-by-line,
extracting article title and "pre-processing" article body
which can then be passed to a Markdown compiler at the call site.

When the scanner encounters an "embed" code fence like this...

```embed
example.rb setup
```

...it loads the source code file at articles/code/example.rb
and finds "magic comments" in the source like this...

# begindoc: setup
puts "here"
# enddoc: setup

The lines between magic comments
are embedded back in the original code fence.

Bad input in the Markdown document or source code file
will cause a panic.
*/
func preProcess(filepath string) (title, body string) {
	f, err := os.Open(filepath)
	check(err)
	defer f.Close()

	var (
		scanner = bufio.NewScanner(f)
		isFirst = true
		isEmbed = false
	)

	for scanner.Scan() {
		line := scanner.Text()

		if isFirst {
			if strings.Index(line, "# ") == -1 {
				panic("error: first line must be an h1 like: # Intro")
			}

			title = line[2:len(line)]
			isFirst = false
			continue
		}

		if line == "```embed" {
			isEmbed = true
			continue
		}

		if isEmbed {
			parts := strings.Split(line, " ")
			if len(parts) != 2 {
				panic("error: embed line must be filename<space>id like: test.rb id")
			}

			var (
				filename = parts[0]
				id       = parts[1]
				lang     = "ruby"
				comment  = "#"
			)

			if path.Ext(filename) != ".rb" {
				panic("error: only .rb files are supported, not " + path.Ext(filename))
			}

			srcCode, err := ioutil.ReadFile(wd + "/articles/code/" + filename)
			check(err)

			sep := comment + " begindoc: " + id + "\n"
			begindoc := strings.Index(string(srcCode), sep)
			if begindoc == -1 {
				panic("error: separator not found " + sep)
			}
			begindoc += len(sep) // begin at end of separator

			sep = comment + " enddoc: " + id
			enddoc := strings.Index(string(srcCode), sep)
			if enddoc == -1 {
				panic("error: separator not found " + sep)
			}
			enddoc-- // adjust for newline

			body += "```" + lang + "\n" + string(srcCode[begindoc:enddoc])

			isEmbed = false
			continue
		}

		body += "\n" + line
	}

	return title, body
}
