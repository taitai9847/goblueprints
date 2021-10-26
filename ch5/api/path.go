package main

import "strings"

const PathSeparator = "/"

type Path struct {
	Path string
	ID   string
}

// ex) /people/1

func NewPath(p string) *Path {
	var id string
	p = strings.Trim(p, PathSeparator)   // people/1
	s := strings.Split(p, PathSeparator) // [people 1]
	if len(s) > 1 {
		id = s[len(s)-1]                              // 1
		p = strings.Join(s[:len(s)-1], PathSeparator) // people
	}
	return &Path{Path: p, ID: id}
}
func (p *Path) HasID() bool {
	return len(p.ID) > 0
}
