# GitHub Issue-based headless CMS for Hugo

A headless CMS for Hugo using GitHub Issues.

Issues are treated as articles.

## Prerequisites

- Go
- Hugo
- GitHub Token

## Installation

### 1. Install this application

```bash
$ go install github.com/Azunyan1111/github-gh-cms@latest
```

### 2. Create Config file

Create a YAML file named ``gic.config.yaml`` and write your credentials.

```yaml
github:
  username: 'your-name'
  repository: 'your-repository'

hugo:
  url:
    images: '/images'
```

### 3. Run

Run this application with your GitHub Access Token

```bash
$ github-gh-cms generate --token="YOUR_GITHUB_TOKEN"
```

If your repository has issues and attached images, they will be exported like this tree.

These directories are compatible with Hugo directory structure, so you can quickly deploy this application to your Hugo site.

```bash
$ tree --dirsfirst
.
├── content
│   └── posts
│       ├── 2004501283.md
│       └── 2006779255.md
├── static
│   └── images
│       ├── 2004501283
│       │   └── 0.png
│       └── 2006779255
│           ├── 0.png
│           ├── 1.png
│           └── 2.png
└── gic.config.yaml
```

### 4. (Optional) Auto commit with GitHub Actions

First, you set GitHub Token to your repository secret. (On your repository screen, go to "Settings" > "Secrets and variables" > "New repository secret")

On this tutorial, I set the Token as "GH_TOKEN".

Next, you write this workflow.

```yaml
name: Go

on:
  push:
    branches: [ "main" ]
  issues:
    types: [reopened, closed]
  pull_request:
    branches: [ "main" ]
permissions: write-all

jobs:

  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21.4'

    - name: Install
      run: go install github.com/Azunyan1111/github-gh-cms@latest

    - name: Generate
      run: github-gh-cms --token=${{ secrets.GH_TOKEN }}

    - name: Auto Commit
      uses: stefanzweifel/git-auto-commit-action@v4
      with:
        commit_message: "ci: :memo: Add new articles"
```

Congratulation.

Your Hugo site will automatically deployed when a Issue is closed or reopened.
