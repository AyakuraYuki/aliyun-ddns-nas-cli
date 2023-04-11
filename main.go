package main

import (
	"errors"
	"fmt"
	"github.com/json-iterator/go/extra"
	"github.com/urfave/cli/v2"
	"log"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	app          Application
	buildVersion = "missing version [git hash]"
)

////////////////////////////////////////////////////////////////////////////////
// global flags
////////////////////////////////////////////////////////////////////////////////

func cliGlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "access-key-id",
			Aliases: []string{"id"},
			Usage:   "Aliyun OpenAPI Access Key ID",
		},
		&cli.StringFlag{
			Name:    "access-key-secret",
			Aliases: []string{"secret"},
			Usage:   "Aliyun OpenAPI Access Key Secret",
		},
		&cli.StringFlag{
			Name:    "ip-api",
			Aliases: []string{"api"},
			Usage:   "Specify API to fetch ip address, e.g. https://v6r.ipip.net/",
		},
		&cli.StringFlag{
			Name:    "ip2location-api-key",
			Aliases: []string{"ip2loc-key", "ip2l"},
			Usage:   "Specify API key for using IP2Location API to get IP GeoLocation",
		},
		&cli.BoolFlag{
			Name:    "ipv6",
			Aliases: []string{"6"},
			Usage:   "IPv6",
		},
	}
}

////////////////////////////////////////////////////////////////////////////////
// application initialization
////////////////////////////////////////////////////////////////////////////////

func initialize(c *cli.Context, validateAccessKey bool) error {
	// Aliyun Client config
	ids := []string{c.String("access-key-id"), os.Getenv("AccessKeyId")}
	secrets := []string{c.String("access-key-secret"), os.Getenv("AccessKeySecret")}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	sort.Sort(sort.Reverse(sort.StringSlice(secrets)))
	app.AccessKeyId = ids[0]
	app.AccessKeySecret = secrets[0]

	// IP2Location config
	ip2LocationApiKeys := []string{c.String("ip2location-api-key"), os.Getenv("IP2LocationApiKey")}
	sort.Sort(sort.Reverse(sort.StringSlice(ip2LocationApiKeys)))
	app.Ip2LocationApiKey = ip2LocationApiKeys[0]

	// Aliyun Client validation
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

	// IPv6
	if c.Bool("ipv6") {
		ipFunc.MyIP = GetIPv6
		ipFunc.Resolver = ResolveIPv6
	}

	// Custom FindMyIP Function
	ipApi := make([]string, 0)
	for _, api := range c.StringSlice("ip-api") {
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

////////////////////////////////////////////////////////////////////////////////
// cli command - list
////////////////////////////////////////////////////////////////////////////////

func cliListDomainRecordsFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "domain",
			Aliases: []string{"d"},
			Usage:   "domain name, e.g. aliyun.com",
		},
	}
}

func cliListDomainRecords(c *cli.Context) error {
	if err := initialize(c, true); err != nil {
		return err
	}

	domainName := c.String("domain")
	if !Contains(app.managedDomainNames, domainName) {
		return fmt.Errorf("domain name not managed: %s", domainName)
	}

	records, err := app.DescribeDomainRecords(domainName)
	if err != nil {
		return err
	}

	for _, v := range records {
		fmt.Printf("%40s   %-16s %s\n", fmt.Sprintf("%s.%s", *v.RR, *v.DomainName), fmt.Sprintf("%6s (TTL:%4d)", *v.Type, *v.TTL), *v.Value)
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// cli command - get-ip
////////////////////////////////////////////////////////////////////////////////

func cliGetIp(c *cli.Context) error {
	if err := initialize(c, false); err != nil {
		return err
	}

	ip := ipFunc.MyIP()
	fmt.Println(ip)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// cli command - resolve
////////////////////////////////////////////////////////////////////////////////

func cliResolveFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "domain",
			Aliases: []string{"d"},
			Usage:   "domain name, e.g. aliyun.com",
		},
	}
}

func cliResolve(c *cli.Context) error {
	if err := initialize(c, false); err != nil {
		return err
	}

	domain := c.String("domain")
	ip := ResolveIPv4(domain)
	fmt.Println(ip)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// cli command - update
////////////////////////////////////////////////////////////////////////////////

func cliUpdateFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "domain",
			Aliases: []string{"d"},
			Usage:   "domain name, e.g. aliyun.com",
		},
		&cli.StringFlag{
			Name:    "ip-address",
			Aliases: []string{"ip"},
			Usage:   "Specify IP address, e.g. 1.2.3.4",
		},
	}
}

func cliUpdate(c *cli.Context) error {
	if err := initialize(c, true); err != nil {
		return err
	}

	domain := c.String("domain")
	rr, domainName, err := app.CheckDomainNameAndRR(SplitDomain(domain))
	if err != nil {
		return err
	}
	recordType := "A"
	if c.Bool("ipv6") {
		recordType = "AAAA"
	}
	ipAddress := c.String("ip-address")
	err = app.CliUpdateRecord(rr, domainName, ipAddress, recordType)
	if err != nil {
		return err
	}
	log.Printf("Updated! domain: %s, IP: %s, record type: %s\n", domain, ipAddress, recordType)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// cli command - delete
////////////////////////////////////////////////////////////////////////////////

func cliDeleteFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "domain",
			Aliases: []string{"d"},
			Usage:   "domain name, e.g. aliyun.com",
		},
	}
}

func cliDelete(c *cli.Context) error {
	if err := initialize(c, true); err != nil {
		return err
	}

	domain := c.String("domain")
	rr, domainName, err := app.CheckDomainNameAndRR(SplitDomain(domain))
	if err != nil {
		return err
	}
	err = app.DeleteDomainRecord(rr, domainName)
	if err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// cli command - auto-update
////////////////////////////////////////////////////////////////////////////////

func cliAutoUpdateFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "domain",
			Aliases: []string{"d"},
			Usage:   "domain name, e.g. aliyun.com",
		},
		&cli.StringFlag{
			Name:    "redo",
			Aliases: []string{"repeat", "re", "r"},
			Usage:   "redo (or repeat) by giving seconds, set 0 to disable auto update (just like command update will do)",
		},
	}
}

func cliAutoUpdate(c *cli.Context) error {
	if err := initialize(c, true); err != nil {
		return err
	}

	domain := c.String("domain")
	rr, domainName, err := app.CheckDomainNameAndRR(SplitDomain(domain))
	if err != nil {
		return err
	}
	recordType := "A"
	if c.Bool("ipv6") {
		recordType = "AAAA"
	}
	redo := c.String("redo")
	if redo != "" && !regexp.MustCompile(`\d+[Rr]?$`).MatchString(redo) {
		return errors.New("wrong format of parameter redo")
	}
	randomDelay := regexp.MustCompile(`\d+[Rr]$`).MatchString(redo)
	redoSeconds := 0
	if randomDelay {
		redoSeconds, _ = strconv.Atoi(redo[:len(redo)-1])
	} else {
		redoSeconds, _ = strconv.Atoi(redo)
	}

	for {
		ipAddress := ipFunc.MyIP()
		if ipAddress == "" {
			log.Printf("cannot get ip address, please check your network")
		} else {
			err = app.CliUpdateRecord(rr, domainName, ipAddress, recordType)
			if err != nil {
				log.Printf("update domain record error cause: %v", err)
			} else {
				log.Printf("(auto) Updated! domain: %s, IP: %s, record type: %s\n", domain, ipAddress, recordType)
			}
		}
		if redoSeconds == 0 {
			break
		}
		if randomDelay {
			time.Sleep(time.Duration(redoSeconds+rand.Intn(redoSeconds)) * time.Second)
		} else {
			time.Sleep(time.Duration(redoSeconds) * time.Second)
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// program init & entrance
////////////////////////////////////////////////////////////////////////////////

const (
	CategoryDDNS     = "DDNS"
	CategoryMaintain = "Maintain"
)

func init() {
	rand.NewSource(time.Now().UnixNano())
	extra.RegisterFuzzyDecoders()
}

func main() {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Println("Err: exit unexpected!", recovered)
		}
	}()

	cliApp := &cli.App{
		Name:           "ali-ddns",
		Usage:          "aliyun-ddns-nas-cli",
		Version:        fmt.Sprintf("Git:[%s] (%s)", strings.ToUpper(buildVersion), runtime.Version()),
		Compiled:       time.Time{},
		Authors:        []*cli.Author{{Name: "Ayakura Yuki"}},
		DefaultCommand: "help",
		Commands: []*cli.Command{
			{
				Name:     "list",
				Aliases:  []string{"l"},
				Category: CategoryDDNS,
				Usage:    "Describe all domain records by giving domain name",
				Flags:    cliListDomainRecordsFlags(),
				Action:   cliListDomainRecords,
			},
			{
				Name:     "get-ip",
				Category: CategoryMaintain,
				Usage:    "Get IP address combines multiple Web-API",
				Action:   cliGetIp,
			},
			{
				Name:     "resolve",
				Category: CategoryMaintain,
				Usage:    "Resolve DNS-IPv4 combines multiple DNS upstreams",
				Flags:    cliResolveFlags(),
				Action:   cliResolve,
			},
			{
				Name:     "update",
				Aliases:  []string{"up", "u"},
				Category: CategoryDDNS,
				Usage:    "Update domain record, or create record if not exists",
				Flags:    cliUpdateFlags(),
				Action:   cliUpdate,
			},
			{
				Name:     "delete",
				Category: CategoryDDNS,
				Usage:    "Delete domain record",
				Flags:    cliDeleteFlags(),
				Action:   cliDelete,
			},
			{
				Name:     "auto-update",
				Aliases:  []string{"auto-up", "auto"},
				Category: CategoryDDNS,
				Usage:    "Update domain record automatically",
				Flags:    cliAutoUpdateFlags(),
				Action:   cliAutoUpdate,
			},
		},
		Flags: cliGlobalFlags(),
		Action: func(c *cli.Context) error {
			return initialize(c, true)
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		panic(err)
	}
}
