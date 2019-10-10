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

	"github.com/kr/jsonfeed"
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
		fmt.Println(err)
		os.Exit(1)
	}
}

func exitWith(s string) {
	fmt.Println(s)
	os.Exit(1)
}

// Blog contains data loaded from config.json
type Blog struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Article contains data loaded from config.json and parsed Markdown files
type Article struct {
	Author        string        `json:"author"`
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
		Author:    u.Name,
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

	dat, err := ioutil.ReadFile(wd + "/theme/_headers")
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
	blog, articles, tagMap := load()

	check(os.RemoveAll(wd + "/public"))
	check(os.MkdirAll(wd+"/public/images", os.ModePerm))
	check(os.MkdirAll(wd+"/public/tags", os.ModePerm))

	tags := []string{}
	tagPage := template.Must(template.ParseFiles(wd + "/theme/tag.html"))
	for tag, tagArticles := range tagMap {
		tags = append(tags, tag)

		f, err := os.Create("public/tags/" + tag + ".html")
		check(err)
		tagData := struct {
			Tag      string
			Blog     Blog
			Articles []Article
		}{
			Tag:      tag,
			Blog:     blog,
			Articles: tagArticles,
		}
		check(tagPage.Execute(f, tagData))
	}

	indexPage := template.Must(template.ParseFiles(wd + "/theme/index.html"))
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

	feed := jsonfeed.Feed{
		Title:       blog.Name,
		HomePageURL: blog.URL,
		FeedURL:     blog.URL + "/feed.json",
	}
	feed.Items = make([]jsonfeed.Item, len(articles))

	articlePage := template.Must(template.ParseFiles(wd + "/theme/article.html"))
	for i, a := range articles {
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

		item := jsonfeed.Item{
			ID:          blog.URL + "/" + a.ID,
			URL:         blog.URL + "/" + a.ID,
			Title:       a.Title,
			ContentHTML: string(a.Body),
			Tags:        a.Tags,
		}
		published, err := time.Parse("2006-01-02", a.Published)
		if err == nil {
			item.DatePublished = published
		}
		updated, err := time.Parse("2006-01-02", a.LastUpdated)
		if err == nil {
			item.DateModified = updated
		}
		item.Author = &jsonfeed.Author{Name: a.Author}
		feed.Items[i] = item
	}

	f, err = os.Create("public/feed.json")
	check(err)
	check(json.NewEncoder(f).Encode(&feed))

	cmd := exec.Command("cp", "-a", wd+"/articles/images/.", wd+"/public/images")
	cmd.Run()

	cmd = exec.Command("cp", wd+"/theme/{_headers,_redirects}", wd+"/public")
	cmd.Run()
}

func load() (Blog, []Article, map[string][]Article) {
	config, err := ioutil.ReadFile(wd + "/config.json")
	check(err)
	var data struct {
		Blog     Blog      `json:"blog"`
		Articles []Article `json:"articles"`
	}
	check(json.Unmarshal(config, &data))

	articles := make([]Article, len(data.Articles))
	tagMap := make(map[string][]Article)

	for i, a := range data.Articles {
		title, body := preProcess("articles/" + a.ID + ".md")
		markdown := blackfriday.Run([]byte(body))

		lastUpdated := a.Published
		if a.Updated != "" {
			lastUpdated = a.Updated
		}
		t, err := time.Parse("2006-01-02", lastUpdated)
		check(err)

		a := Article{
			Author:        a.Author,
			Body:          template.HTML(markdown),
			Canonical:     a.Canonical,
			ID:            a.ID,
			LastUpdated:   lastUpdated,
			LastUpdatedIn: t.Format("2006 January"),
			LastUpdatedOn: t.Format("January 2, 2006"),
			Published:     a.Published,
			Tags:          a.Tags,
			Title:         title,
			Updated:       a.Updated,
		}
		articles[i] = a

		for _, t := range a.Tags {
			tagMap[t] = append(tagMap[t], a)
		}
	}

	return data.Blog, articles, tagMap
}

/*
preProcess scans the Markdown document at filepath line-by-line,
extracting article title and "pre-processing" article body
which can then be passed to a Markdown compiler at the call site.

When the scanner encounters an "embed" code fence like this...

```embed
example.rb id
```

...it loads the source code file at articles/code/example.rb
and finds "magic comments" in the source like this...

# begindoc: id
puts "here"
# enddoc: id

The lines between magic comments
are embedded back in the original code fence.

Bad input in the Markdown document or source code file
will stop the program with a non-zero exit code and error text.
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
				exitWith("error: first line must be an h1 like: # Intro")
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
				exitWith("error: embed line must be filename<space>id like: test.rb id")
			}
			filename := parts[0]
			id := parts[1]

			srcCode, err := ioutil.ReadFile(wd + "/articles/code/" + filename)
			check(err)

			sep := "begindoc: " + id + "\n"
			begindoc := strings.Index(string(srcCode), sep)
			if begindoc == -1 {
				exitWith("error: embed separator not found " + sep + " in " + filename)
			}
			// end of comment line
			begindoc += len(sep)

			sep = "enddoc: " + id
			enddoc := strings.Index(string(srcCode), sep)
			if enddoc == -1 {
				exitWith("error: embed separator not found " + sep + " in " + filename)
			}
			// backtrack to last newline to cut out comment character(s)
			enddoc = strings.LastIndex(string(srcCode[0:enddoc]), "\n")

			var lines []string
			for _, l := range strings.Split(string(srcCode[begindoc:enddoc]), "\n") {
				lines = append(lines, strings.TrimSpace(l))
			}

			ext := strings.Trim(path.Ext(filename), ".")
			body += "```" + ext + "\n" + strings.Join(lines, "\n")

			isEmbed = false
			continue
		}

		body += "\n" + line
	}

	return title, body
}
