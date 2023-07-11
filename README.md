# postchi
Yet another HTTP client that doesn't SUCK.



## Sample config file
```yaml
vars:
  baseURL:
    source: process
    name: BASE_URL
    value: http://localhost:1323/api/v2/asset
  jwtToken:
    source: process
    name: JWT_TOKEN
    value: ""
  assetID:
    source: args
    index: 0

defaults:
  headers:
    Authorization: "Bearer {{ .jwtToken }}"
    Content-Type: "application/json"
    Accepts: "application/json"

requests:
  asset:
    route: "{{ .baseURL }}/{{ .assetID }}"
  feeds:
    route: "{{ .baseURL }}/{{ .assetID }}/feed"

```

## Usage
```bash
postchi -file postchi.yaml -name feeds
postchi -name feeds #defaults to using postchi.yaml in your current PWD
```
I recommend having `jq` installed on your system and use this tool in combination of that and a text editor, so you can analyze output of the api more easily.
```bash
postchi -name index someid | jq | code -
postchi -name index someid | jq | vim
```
### Interactive mode
```bash
postchi -interactive # will open up your $EDITOR and you write your request in HTTP format
```
