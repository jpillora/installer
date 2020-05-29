
# `installer`

Quickly install pre-compiled binaries from Github releases.

Installer is an HTTP server which returns shell scripts. The returned script will detect platform OS and architecture, choose from a selection of URLs, download the appropriate file, un(zip|tar|gzip) the file, find the binary (largest file) and optionally move it into your `PATH`. Useful for installing your favourite pre-compiled programs on hosts using only `curl`.

## Usage

```sh
# install <user>/<repo> from github
curl https://i.jpillora.com/<user>/<repo>@<release>! | bash
```

```sh
# search Google for github repo <query>
curl https://i.jpillora.com/<query>! | bash
```

*Or you can use* `wget -qO- <url> | bash`

**Path API**

* `user` Github user (defaults to @jpillora, customisable if you [host your own](#host-your-own), uses Google to pick most relevant `user` when `repo` not found)
* `repo` Github repository belonging to `user` (**required**)
* `release` Github release name (defaults to the **latest** release)
* `!` When provided, downloads binary directly into `/usr/local/bin/` (defaults to working directory)
* `!!` Uses `sudo` to `mv` into `/usr/local/bin/`

**Query Params**

* `?type=` Force the return type to be one of: `script` or `homebrew`
    * `type` is normally detected via `User-Agent` header
    * `type=homebrew` is **not** working at the moment – see [Homebrew](#homebrew)
* `?insecure=1` Force `curl`/`wget` to skip certificate checks

## Security

:warning: Although I promise [my instance of `installer`](https://i.jpillora.com/) is simply a copy of this repo - you're right to be wary of piping shell scripts from unknown servers, so you can host your own server [here](#host-your-own) or just leave off `| bash` and checkout the script yourself.

## Examples

* https://i.jpillora.com/serve
* https://i.jpillora.com/cloud-torrent
* https://i.jpillora.com/yudai/gotty@v0.0.12
* https://i.jpillora.com/mholt/caddy
* https://i.jpillora.com/caddy
* https://i.jpillora.com/rclone

    ```sh
    $ curl -s i.jpillora.com/mholt/caddy! | bash
    Downloading mholt/caddy v0.8.2 (https://github.com/mholt/caddy/releases/download/v0.8.2/caddy_darwin_amd64.zip)
    ######################################################################## 100.0%
    Downloaded to /usr/local/bin/caddy
    $ caddy --version
    Caddy 0.8.2
    ```

## Host your own

* Install installer with installer

    ```sh
    curl -s https://i.jpillora.com/installer | bash
    ```

* Install from source

    ```sh
    go get github.com/jpillora/installer
    ```

* Install on Heroku

    Click this button to deploy for free on [Heroku](https://heroku.com)

	[![Deploy](https://www.herokucdn.com/deploy/button.png)](https://heroku.com/deploy)

    *You can optionally add your own domain as a app custom domain.*

### Homebrew

Currently, installing via Homebrew does not work. Homebrew was intended to be supported with:

```
#does not work
brew install https://i.jpillora.com/serve
```

However, homebrew formulas require an SHA1 hash of each binary and currently, the only way to get is to actually download the file. It **might** be acceptable to download all assets if the resulting `.rb` file was cached for a long time.

#### MIT License

Copyright © 2020 Jaime Pillora &lt;dev@jpillora.com&gt;

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
