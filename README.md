# WIP: botassasin

## Configuration
| Param                | Type          | Description
|----------------------|---------------|----------------------------
| debug                | bool          | Print more information
| logfile              | string        | File watched by botassasin
| log_format           | string        | Line format in logfile. Must be regexp in [Go re2 syntax](https://github.com/google/re2/wiki/Syntax) (ex. `^(?P<ip>\d+\.\d+\.\d+\.\d+) - - \[.{26}\] \"(?P<request>[^\"]*)\" \d{3} \d+ \"(?P<referer>[^\"]*)\" \"(?P<user_agent>[^\"]*)\" rt.*$`)
| checkers             | array         | List of checkers with configuration. Checkers executed in order
| block_action         | string\|array | Command used for block bot when checkers say so. If command should accept params array syntax must be used. Can be used syntax of `text/template` package. Supported params in template is `{{.ip}}` and named capture groups from `log_format`<br>Ex.
```yaml
block_action:
 - ip
 - route
 - add
 - blackhole
 - "{.ip}}"
```
| blocklog             | string        | Block action log file. 
| blocklog_template    | string        | Format used for `blocklog`. Can be used syntax of `text/template` package. Supported params in template is `{{.ip}}`, `{{.time}}` and named capture groups from `log_format`. Also checkers can add their own params like `geoip` add `{{.country}}` param. Ex. `{{.time}} {{.ip}} {{.country}} {{.checker}} "{{.user_agent}}" "{{.referer}}"`
| whitelist_cache_path | string        | Whitelist cache file. Drop cache to disc every minute. On next run load whitelisted ip from disk