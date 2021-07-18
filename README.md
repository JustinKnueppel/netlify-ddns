# netlify-ddns

Dynamic DNS implementation using Netlify's API

## Usage

`nddns -domain example.com -pat <netlify_pat> [-subdomain home.ddns] [-ttl 300] [-poll 30m]`

## Arguments

| Name        | Type       | Default      | Required           | Description                                                                                        |
| ----------- | ---------- | ------------ | ------------------ | -------------------------------------------------------------------------------------------------- |
| `domain`    | `string`   | `nil`        | :heavy_check_mark: | Domain that is registered with Netlify                                                             |
| `subdomain` | `string`   | `nil`        | :x:                | Subdomain to which A record will be set                                                            |
| `pat`       | `string`   | `$NDDNS_PAT` | :x:                | Personal access token for Netlify. Must be set as environment variable `NDDNS_PAT` if not supplied |
| `ttl`       | `int`      | `300`        | :x:                | TTL for DNS record                                                                                 |
| `poll`      | `duration` | `30m`        | :x:                | Interval at which to poll for changes to local IP                                                  |
