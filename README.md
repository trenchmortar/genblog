# genblog

A static blog generator featuring:

* No external dependencies
* Markdown files with no front matter
* Local preview server
* JSON feed
* Images
* Drafts
* Anonymous, single author, or multiple authors
* Tags
* "Last updated" timestamp
* Redirects
* `rel=canonical` tags
* Default theme (can be edited directly)

The default theme features:

* Responsive design
* PageSpeed Insights performance score of 100
* Mozilla Observatory security grade of A+

[See a blog](https://www.statusok.com)
modified slightly from the default theme.

## Create a blog

Download the latest release:

```
curl -sL https://github.com/statusok/genblog/releases/download/v0.1.0/blog.tar.gz | tar xvz
```

It contains:

```
.
├── .gitignore
├── README.md
├── articles
│   ├── code
│   └── images
├── bin
│   ├── linux
│   └── mac
├── config.json
├── genblog
├── public
└── theme
    ├── _headers
    ├── article.html
    ├── index.html
    └── tag.html
```

The `./genblog` script invokes the OS-specific `./bin/{linux,mac}` binary.
It requires no external dependencies (such as programming languages).

## Write

Add an article:

```
./genblog add example-article
```

Edit `articles/example-article.md` in a text editor.
This is a pure Markdown file with no front matter.

The first line of the file is the article title.
It must be an `<h1>` tag:

```md
# Example Article
```

Preview at <http://localhost:2000> with:

```
./genblog serve
```

See the [JSON feed](https://jsonfeed.org/) at <http://localhost:2000/feed.json>.

Add images to the `articles/images` directory.
Refer to them in articles via relative path:

```md
![alt text](images/example.png)
```

## Configure

Configure blog in `config.json`:

```json
{
  "blog": {
    "name": "Blog",
    "url": "https://blog.example.com"
  },
  "articles": [
    {
      "id": "article-is-draft-if-published-is-future-date",
      "published": "2050-01-01"
    },
    {
      "id": "article-with-anonymous-author",
      "published": "2018-04-15"
    },
    {
      "author": "Alice",
      "id": "article-with-single-author",
      "published": "2018-04-01"
    },
    {
      "author": "Alice and Bob",
      "id": "article-with-multiple-authors",
      "published": "2018-03-15"
    },
    {
      "id": "article-with-tags",
      "published": "2018-03-01",
      "tags": [
        "go",
        "unix"
      ]
    },
    {
      "id": "article-with-updated-date",
      "published": "2018-02-15",
      "updated": "2018-02-20"
    },
    {
      "id": "article-with-redirects",
      "published": "2018-02-01",
      "redirects": [
        "/article-original-name",
        "/article-renamed-again",
        "/this-feature-works-only-on-netlify",
      ]
    },
    {
      "canonical": "https://seo.example.com/avoid-duplicate-content-penalty",
      "id": "article-with-rel-canonical",
      "published": "2018-01-15"
    }
  ]
}
```

## Extend theme

The `theme` directory's files can be edited
to customize the blog's HTTP headers, HTML, CSS, and JavaScript.

```
.
├── _headers
├── article.html
├── index.html
└── tag.html
```

The `_headers` file is copied to `public/_headers` to be used as
[Netlify Headers](https://www.netlify.com/docs/headers-and-basic-auth/).

The `.html` files are used as templates by `./genblog`.
They are parsed as [Go templates](https://gowebexamples.com/templates/).

The `article.html` file accepts a data structure like this:

```
{
  Blog: {
    Name: "Blog",
    URL:  "https://blog.example.com",
  }
  Article: {
    Author:        "Alice",
    Body:          "<p>Hello, world.</p>",
    Canonical:     "https://seo.example.com/avoid-duplicate-content-penalty"
    ID:            "example-article",
    LastUpdated:   "2018-04-15",
    LastUpdatedIn: "2018 April",
    LastUpdatedOn: "April 15, 2018",
    Published:     "2018-04-10",
    Tags:          ["go", "unix"],
    Title:         "Example Article",
    Updated:       "2018-04-15",
  }
}
```

The `index.html` file accepts a data structure like this:

```
{
  Blog: {
    Name: "Blog",
    URL:  "https://blog.example.com",
  },
  Articles: [
    {
      Author:        "Alice",
      Body:          "<p>Hello, world.</p>",
      Canonical:     "https://seo.example.com/avoid-duplicate-content-penalty"
      ID:            "example-article",
      LastUpdated:   "2018-04-15",
      LastUpdatedIn: "2018 April",
      LastUpdatedOn: "April 15, 2018",
      Published:     "2018-04-10",
      Tags:          ["go", "unix"],
      Title:         "Example Article",
      Updated:       "2018-04-15",
    }
  ],
  Tags: ["go", "unix"],
}
```

The `tag.html` file accepts a data structure like this:

```
{
  Blog: {
    Name: "Blog",
    URL:  "https://blog.example.com",
  },
  Articles: [
    {
      Author:        "Alice",
      Body:          "<p>Hello, world.</p>",
      Canonical:     "https://seo.example.com/avoid-duplicate-content-penalty"
      ID:            "example-article",
      LastUpdated:   "2018-04-15",
      LastUpdatedIn: "2018 April",
      LastUpdatedOn: "April 15, 2018",
      Published:     "2018-04-10",
      Tags:          ["go", "unix"],
      Title:         "Example Article",
      Updated:       "2018-04-15",
    }
  ],
  Tag: "go"
}
```

## Publish

Configure [Netlify](https://www.netlify.com):

* Repository: `https://github.com/example/example`
* Branch: `master`
* Build Cmd: `./genblog build`
* Public folder: `public`

To publish articles, commit and push to the GitHub repo.

View deploy logs in the Netlify web interface.

## Get help

Read [MIT License][license].
View [releases].
Open a [GitHub Issue][issues].

[license]: https://github.com/statusok/genblog/blob/master/LICENSE
[releases]: https://github.com/statusok/genblog/releases
[issues]: https://github.com/statusok/genblog/issues

## Contribute

Get a working [Go installation](http://golang.org/doc/install).
For example, on macOS:

```
gover="1.13"

if ! go version | grep -Fq "$gover"; then
  sudo rm -rf /usr/local/go
  curl "https://dl.google.com/go/go$gover.darwin-amd64.tar.gz" | \
    sudo tar xz -C /usr/local
fi
```

Fork the repo.
Clone the project.

```
git clone https://github.com/statusok/genblog.git
cd genblog
go test -race ./...
```

Make changes.
Push to your fork.
Open a pull request.
Discuss with reviewers.
A maintainer will merge.
