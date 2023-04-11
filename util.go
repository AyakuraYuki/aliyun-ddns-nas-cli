package main

import (
	"fmt"
	"strings"
)

import "github.com/json-iterator/go"

var JSON = jsoniter.ConfigCompatibleWithStandardLibrary

type IpFunc struct {
	MyIP     func() string
	Resolver func(domain string) string
}

var ipFunc = IpFunc{
	MyIP:     GetIPv4,
	Resolver: ResolveIPv4,
}

func MaskString(s string) string {
	if s == "" {
		return ""
	}
	stars := len(s) - 6
	if stars <= 0 {
		return strings.Repeat(`*`, len(s))
	} else if stars < 4 {
		return fmt.Sprintf("%s%s%s", s[:1], strings.Repeat(`*`, len(s)-2), s[len(s)-1:])
	}
	head := s[:3]
	last := s[len(s)-3:]
	return fmt.Sprintf("%s%s%s", head, strings.Repeat(`*`, stars), last)
}

func Contains(slice []string, key string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	_, ok := set[key]
	return ok
}
