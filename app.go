package main

import (
	"fmt"
	alidns20150109 "github.com/alibabacloud-go/alidns-20150109/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"log"
	"strings"
)

type Application struct {
	AccessKeyId          string
	AccessKeySecret      string
	alidns20150109Client *alidns20150109.Client
	managedDomainNames   []string
	Ip2LocationApiKey    string
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
		log.Printf("DescribeDomains error cause: %v\n", err)
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
			log.Printf("DescribeDomainRecords error cause: %v\n", err0)
			return records, err0
		}
		records = append(records, resp.Body.DomainRecords.Record...)
		if int64(len(records)) >= *resp.Body.TotalCount {
			break
		}
	}
	return records, nil
}

func (app *Application) AddDomainRecord(rr, domainName, recordType, value string) error {
	resp, err := app.client().AddDomainRecord(&alidns20150109.AddDomainRecordRequest{
		DomainName: tea.String(domainName),
		RR:         tea.String(rr),
		Type:       tea.String(recordType),
		Value:      tea.String(value),
		TTL:        tea.Int64(600),
	})
	if err != nil {
		log.Printf("AddDomainRecord error cause: %v\n", err)
		return err
	}
	log.Printf("AddDomainRecord successed, resp: %v", resp)
	return nil
}

func (app *Application) UpdateDomainRecord(recordId, rr, recordType, value string) error {
	resp, err := app.client().UpdateDomainRecord(&alidns20150109.UpdateDomainRecordRequest{
		RecordId: tea.String(recordId),
		RR:       tea.String(rr),
		Type:     tea.String(recordType),
		Value:    tea.String(value),
		TTL:      tea.Int64(600),
	})
	if err != nil {
		log.Printf("UpdateDomainRecord error cause: %v\n", err)
		return err
	}
	log.Printf("UpdateDomainRecord successed, resp: %v", resp)
	return nil
}

func (app *Application) DeleteDomainRecord(rr, domainName string) error {
	records, err := app.DescribeDomainRecords(domainName)
	if err != nil {
		return err
	}
	for _, record := range records {
		if *record.RR != rr {
			continue
		}
		resp, err := app.client().DeleteDomainRecord(&alidns20150109.DeleteDomainRecordRequest{RecordId: record.RecordId})
		if err != nil {
			log.Printf("DeleteDomainRecord error cause: %v\n", err)
			return err
		}
		log.Printf("DeleteDomainRecord successed, resp: %v", resp)
	}
	return nil
}

func (app *Application) CliUpdateRecord(rr, domainName, ipAddress, recordType string) (err error) {
	domain := strings.Join([]string{rr, domainName}, `.`)
	if ipFunc.Resolver(domain) == ipAddress {
		return // skip by same ip address
	}

	recordAmount := 0
	records, err := app.DescribeDomainRecords(domainName)
	if err != nil {
		log.Printf("CliUpdateRecord error cause: %v\n", err)
		return err
	}

	var target *alidns20150109.DescribeDomainRecordsResponseBodyDomainRecordsRecord
	for _, record := range records {
		if *record.RR == rr && *record.Type == recordType {
			target = record
			recordAmount++
		}
	}

	if recordAmount > 1 {
		// 找到重复的解析（可能是多地区），删除所有已存在的记录（因为不需要人工维护）
		_ = app.DeleteDomainRecord(rr, domainName)
		target = nil
	}

	if target == nil {
		// 重新提交解析
		err = app.AddDomainRecord(rr, domainName, recordType, ipAddress)
	} else if *target.Value != ipAddress {
		// 更新解析
		if *target.Type != recordType {
			return fmt.Errorf("record type error (old type: %v, new type: %v)", target.Type, recordType)
		}
		err = app.UpdateDomainRecord(*target.RecordId, *target.RR, *target.Type, ipAddress)
	}

	if err != nil && strings.Contains(err.Error(), `DomainRecordDuplicate`) {
		// 遇到重复的解析记录，删除记录并进入递归重新走更新流程
		_ = app.DeleteDomainRecord(rr, domainName)
		return app.CliUpdateRecord(rr, domainName, ipAddress, recordType)
	}

	return err
}
