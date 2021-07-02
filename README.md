# botassasin



## Configuration

botassasin require `config.yml` for run. Check `config.yml.example` for full example

| Param                | Type          | Description
|----------------------|---------------|----------------------------
| debug                | bool          | Print more information
| logfile              | string        | File watched by botassasin
| log_format           | string        | Line format in logfile. Must be regexp in [Go re2 syntax](https://github.com/google/re2/wiki/Syntax) (ex. `^(?P<ip>\d+\.\d+\.\d+\.\d+) - - \[.{26}\] \"(?P<request>[^\"]*)\" \d{3} \d+ \"(?P<referer>[^\"]*)\" \"(?P<user_agent>[^\"]*)\" rt.*$`)
| checkers             | array         | List of checkers with configuration. Checkers executed in order
| block_action         | string\|array | Command used for block bot when checkers say so. If command should accept params array syntax must be used. Can be used syntax of `text/template` package. Supported params in template is `{{.ip}}` and named capture groups from `log_format`
| blocklog             | string        | Block action log file. 
| blocklog_template    | string        | Format used for `blocklog`. Can be used syntax of `text/template` package. Supported params in template is `{{.ip}}`, `{{.time}}` and named capture groups from `log_format`. Also checkers can add their own params like `geoip` add `{{.country}}` param. Ex. `{{.time}} {{.ip}} {{.country}} {{.checker}} "{{.user_agent}}" "{{.referer}}"`
| whitelist_cache_path | string        | Whitelist cache file. Drop cache to disk every minute. On next run whitelist will be loaded from disk

## Checkers

### list
Blacklist or whitelist of servers. IPv4 or IPv4 with mask is supported (ex. `192.168.1.1`, `192.168.1.2/16`)

Example
```yaml
checkers:
  - kind: list
    sources:
    - src: ./lists/our_servers.txt
      type: txt
      action: whitelist
    - src: https://ip-ranges.amazonaws.com/ip-ranges.json
      type: aws_ip_ranges
      action: whitelist
      aws_service_filter:
        - ROUTE53_HEALTHCHECKS
    - src: https://check.torproject.org/torbulkexitlist
      type: txt
      action: block
```
General params

| Param    | Type   | Description
|----------|--------|-----------------
| kind     | string | Kind of checker, always must be `list`
| sources  | array  | List of ip sources

Sources params
| Param    | Type   | Description
|----------|--------|-----------------
| src      | string | Source of list, can be local path (ex. `./whitelist.txt`) or remote URL (ex. `https://check.torproject.org/torbulkexitlist`)
| type     | string | Format of list: `txt`, `aws_ip_ranges`. `txt` format is single IPv4 or IPv4 with mask for line, comments started with `#` is supported. `aws_ip_ranges` is json provided by AWS https://ip-ranges.amazonaws.com/ip-ranges.json
| aws_service_filter | array | Filters by service, only used with `aws_ip_ranges` source (ex. ROUTE53_HEALTHCHECKS)
| action | stirng | Action when IP match list: `whitelist`, `block`

### field

Checks if field contains substring. Fields is named capture groups in `log_format`

Example
```yml
- kind: field
  field_name: user_agent
  contains:
    - python-requests
  action: block
```

General params

| Param      | Type   | Description
|------------|--------|-----------------
| kind       | string | Kind of checker, always must be `field`
| field_name | string | Field for search substring (ex. `user_agent`)
| contains   | array  | List of substrings for search
| action     | string | Action when field contain one of substrings: `whitelist`, `block`

### geoip

Whitelist by country. GeoLite2-Country database is used 

Eample
```yaml
- kind: geoip
  allowed_countries:
    - RU
  path: ""
```
General params

| Param      | Type   | Description
|------------|--------|-----------------
| kind       | string | Kind of checker, always must be `geoip`
| allowed_countries | array | List of whitelisted countries, other countries will be banned 
| path       | string | Local path to geoip2 database, if empty then embeded database will be used

### reverse_dns

Reverse DNS checker. Mainly used for verify search engines bots.
1. Make reverse DNS query via `resolver` and get hostname
2. Make sure hostname ends with `domain_suffixes`
3. Make forward DNS query and verify hostname resolve to original IP

Example
```yaml
- kind: reverse_dns
  rules:
    - field: user_agent
      field_contains:
        - Google
        - Googlebot
        - googleweblight
      domain_suffixes: 
        - googlebot.com
        - google.com
      resolver: 8.8.8.8:53
```

General params

| Param      | Type   | Description
|------------|--------|-----------------
| kind       | string | Kind of checker, always must be `reverse_dns`
| rules | array | List of rules for verify bots

Rule params
| Param      | Type   | Description
|------------|--------|-----------------
| field       | string | Field for check. Rule triggered if field contains specified substrins
| field_contains | array | List of substrings. One of those substring must be present in `field` for trigger rule
| domain_suffixes | array | List of suffixes. Hostname of reverse DNS query must have one of suffixes othervise IP will be banned
| resolver | string | Address of DNS server if empty use system resolver