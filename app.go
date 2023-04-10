package main

import (
	"fmt"
	alidns20150109 "github.com/alibabacloud-go/alidns-20150109/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"strings"
)

type Application struct {
	AccessKeyId          string
	AccessKeySecret      string
	alidns20150109Client *alidns20150109.Client
	managedDomainNames   []string
}

func (app *Application) String() string {
	return fmt.Sprintf("Application [ AccessKeyId: %s, AccessKeySecret: %s ]", app.AccessKeyId, MaskString(app.AccessKeySecret))
}

func (app *Application) CheckDomainNameAndRR(rr, domainName string) (r, d string, err error) {
	if Contains(app.managedDomainNames, domainName) {
		return rr, domainName, nil
	}

	if !strings.Contains(rr, `.`) {
		return "", "", fmt.Errorf("missing managed record: [%s.%s]", rr, domainName)
	}

	rrSlice := strings.Split(rr, `.`)
	for i := len(rrSlice) - 1; i > 0; i-- {
		d = strings.Join(append(rrSlice[i:], domainName), `.`)
		if Contains(app.managedDomainNames, d) {
			r = strings.Join(rrSlice[:i], `.`)
			return
		}
	}

	return "", "", fmt.Errorf("missing managed record: [%s.%s]", rr, domainName)
}

func (app *Application) client() *alidns20150109.Client {
	if app.AccessKeyId == "" || app.AccessKeySecret == "" {
		return nil
	}
	if app.alidns20150109Client != nil {
		return app.alidns20150109Client
	}
	config := &openapi.Config{}
	config.SetAccessKeyId(app.AccessKeyId)
	config.SetAccessKeySecret(app.AccessKeySecret)
	config.Endpoint = tea.String("alidns.cn-hangzhou.aliyuncs.com")
	var err error
	app.alidns20150109Client, err = alidns20150109.NewClient(config)
	if err != nil {
		panic(err)
	}
	return app.alidns20150109Client
}

func (app *Application) DescribeDomains() (domainNames []string, err error) {
	domainNames = make([]string, 0)
	resp, err := app.client().DescribeDomains(&alidns20150109.DescribeDomainsRequest{
		PageSize: tea.Int64(100),
	})
	if err != nil {
		return
	}
	for _, domain := range resp.Body.Domains.Domain {
		domainNames = append(domainNames, *domain.DomainName)
	}
	return
}

func (app *Application) DescribeDomainRecords(domainName string) (records []*alidns20150109.DescribeDomainRecordsResponseBodyDomainRecordsRecord, err error) {
	records = make([]*alidns20150109.DescribeDomainRecordsResponseBodyDomainRecordsRecord, 0)
	for {
		resp, err0 := app.client().DescribeDomainRecords(&alidns20150109.DescribeDomainRecordsRequest{
			DomainName: tea.String(domainName),
			PageSize:   tea.Int64(500),
		})
		if err0 != nil {
			return records, err0
		}
		records = append(records, resp.Body.DomainRecords.Record...)
		if int64(len(records)) >= *resp.Body.TotalCount {
			break
		}
	}
	return records, nil
}
