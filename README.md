# waybackcollector
Fetch wayback machine historical content for a given url

This tool is basically a wrapper for [Wayback CDX Server API - BETA](https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server) and covers some of it's functionalities.

It allows you to filter and fetch all response content, from archive.org's wayback machine, for a given URL.
What it's also useful for, is that it has a few optimisations for preventing being rate limited by the server so often.

## Install
The tool is written in Go and you can install it using:

```
go get -u github.com/ctoyan/waybackcollector
```

Or you can [download a binary](https://github.com/ctoyan/waybackcollector/releases).

## The problem with CDX
Before you start using this tool, there is an important thing you should know. The CDX server **rate limits your requests**.
If you make too many requests it'll return `429 Too Many Requests` and tell you that you can make max 15 requests per minute.
The interesting thing is that even though it rate limits you, it still allows you to do some requests(dependint on how many per minute you mak) and blocks others.
If you make too much, it'll just block you completely for some amount of time, so you need to find the golden number of requests per minute for your total amount of requests.

Check the params you can use with this tool and try to filter, limit and collapse to get the make the least requests you can.

## Custom params
By custom params, I mean params, that are implemented by the tool(not the CDX server), which allow you to handle the responses in flexible ways.

These are:
- `req-per-sec` - specify how many requests per second you want to make. The problem here is that you can get rate limited quite easily
- `unique` - print to stdout only unique responses
- `print-urls` - only print the request urls. This way you can request them with a tool of your liking
- `output-folder` - save output to a folder. Each response is saved in a checksumed file in that folder
- `time` - show an estimation about how much time it will take to make the requests
- `log-fail-file` - a file to log all failed request urls. This is useful if you want to collect the rate limited request urls
- `log-success-file` - a file to log all failed request urls. This is useful if you want to collect the rate limited request urls

## Basic usage
The only required parameter is `url`. This way the CTX server will return all wayback machine captures for that `url`.

```
$ go run . -url google.com/robots.txt                                                                                                                                          1 ↵ ✹
<head><script src="//archive.org/includes/analytics.js?v=cf34f82" type="text/javascript"></script>
<script type="text/javascript">window.addEventListener('DOMContentLoaded',function(){var v=archive_analytics.values;v.service='wb';v.server_name='wwwb-app101.us.archive.org';v.server_ms=1752;archive_analytics.send_pageview({});});</script><script type="text/javascript" src="/_static/js/ait-client-rewrite.js?v=R-6NEOHA" charset="utf-8"></script>
...
<title>Error response</title>
</head>
<body>
<h1>Error response</h1>
<p>Error code 404.
<p>Message: Not Found.
</body>
...
```

This command returns one response per second and as you can see returns all type of status codes, not just 200.

## Recommended usage
If you want to learn more about the ways you can filter, collapse or limit, you can check the [CDX server docs](https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#intro-and-usage).

The goal is to filter as much as you can and lower the amount of requests you make. An example of such a command is:

```
$ waybackcollector -url google.com/robots.txt -unique -filter statuscode:200 -collapse digest -from 2020 -limit 100 -req-per-sec 5 -log-fail-file fails.log
User-agent: *
Disallow: /search
Allow: /search/about
Allow: /search/static
Allow: /search/howsearchworks
Disallow: /sdch
Disallow: /groups
Disallow: /index.html?
Disallow: /?
Allow: /?hl=
Disallow: /?hl=*&
...

# AdsBot
User-agent: AdsBot-Google
Disallow: /maps/api/js/
Allow: /maps/api/js
Disallow: /maps/api/place/js/
Disallow: /maps/api/staticmap
Disallow: /maps/api/streetview

# Certain social media sites are whitelisted to allow crawlers to access page markup when links to google.com/imgres* are shared. To learn more, please contact images-robots-whitelist@google.com.
User-agent: Twitterbot
Allow: /imgres

User-agent: facebookexternalhit
Allow: /imgres
...
```

This will:
- only print the unique responses (kinda like `output-path` param, without savin into a file)
- print to stdout only 200 responses for that URL ([filter](https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#filtering))
- group together requests by digest ([collapse](https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#collapsing) significantly decreases the number of requests made, but you might miss something if the resource you're searching for is updated frequently)
- history of the file from the beginning of 2020 ([filter](https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#filtering))
- show only 100 responses starting (note that if used with the `unique` param, this will not show 100 unique responses, but the unique responses out of the 100 fetched)
- make 5 requests per second

You can also

## Help output
```
$ waybackcollector -help                                                                                                                                                     1 ↵ ✹ ✭
Usage of waybackcollector:
  -url string
    	URL pattern to collect responses for
  -collapse string
    	A form of filtering, with which you can collaps adjasent fields(find more here: https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#collapsing)
  -filter string
    	Filter your search, using the wayback cdx filters (find more here: https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#filtering)
  -from string
    	Date on which to start collecing responses. Inclusive. Format: yyyyMMddhhmmss. Defaults to first ever record.
  -to string
    	Date on which to end collecing responses. Inclusive. Format: yyyyMMddhhmmss. Defaults to last ever record.
  -limit int
    	Limit the results
  -output-folder string
    	Path to a folder where the tool will safe all unique responses in uniquely named files per response (meg style output)
  -print-urls
    	Print to stdout only a list of historic URLs, which you can request later
  -unique
    	Print to stdout only unique reponses from all fetched
  -log-fail-file
      Path to log file. Log failed requests only
  -log-success-file
      Path to log file. Log successful request urls only
  -time
    	Show roughly how much time it would take to make all requests for the current query
  -verbose
    	Show more detailed output
 ```
