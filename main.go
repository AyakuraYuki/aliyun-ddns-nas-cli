package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	app           Application
	VersionString = "missing version [git hash]"
)

func init() {
	rand.NewSource(time.Now().UnixNano())
}

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = "ali-ddns"
	cliApp.Usage = "aliyun-ddns-nas-cli"
	cliApp.Version = fmt.Sprintf("Git:[%s] (%s)", strings.ToUpper(VersionString), runtime.Version())

	cliApp.Flags = cliFlags()

	// TODO define app.Commands

	cliApp.Action = func(c *cli.Context) error {
		return initialize(c, true)
	}

	if err := cliApp.Run(os.Args); err != nil {
		panic(err)
	}
}

func cliFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{Name: "access-key-id, id", Usage: "Aliyun OpenAPI Access Key ID"},
		cli.StringFlag{Name: "access-key-secret, secret", Usage: "Aliyun OpenAPI Access Key Secret"},
		cli.StringSliceFlag{Name: "ip-api, api", Usage: "Specify API to fetch ip address, e.g. https://v6r.ipip.net/"},
	}
}

func initialize(c *cli.Context, validateAccessKey bool) error {
	ids := []string{c.GlobalString("access-key-id"), os.Getenv("AccessKeyId")}
	secrets := []string{c.GlobalString("access-key-secret"), os.Getenv("AccessKeySecret")}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	sort.Sort(sort.Reverse(sort.StringSlice(secrets)))
	app.AccessKeyId = ids[0]
	app.AccessKeySecret = secrets[0]

	if validateAccessKey {
		if app.client() == nil {
			_ = cli.ShowAppHelp(c)
			return errors.New("aliyun ddns sdk cannot be initialized")
		}
		if domainNames, err := app.DescribeDomains(); err == nil {
			app.managedDomainNames = domainNames
		} else {
			_ = cli.ShowAppHelp(c)
			return errors.New("no more managed domain names")
		}
	}

	ipApi := make([]string, 0)
	for _, api := range c.GlobalStringSlice("ip-api") {
		if !regexp.MustCompile(`^https?://.*`).MatchString(api) {
			api = fmt.Sprintf("https://%s", api)
		}
		if regexp.MustCompile(`(https?)://[-A-Za-z0-9+&@#/%?=~_|!:,.;]+[-A-Za-z0-9+&@#/%=~_|]`).MatchString(api) {
			ipApi = append(ipApi, api)
		}
	}
	if len(ipApi) > 0 {
		re := regexp.MustCompile(RegexIPv4)
		ipFunc.MyIP = func() string {
			return GetIpWithValidator(ipApi, func(s string) string {
				return re.FindString(s)
			})
		}
	}

	return nil
}
