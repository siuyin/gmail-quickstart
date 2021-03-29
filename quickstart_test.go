package main

import (
	"fmt"
	"strings"
	"testing"
)

func Example_mail() {
	m := newEmail("from@b.c", "to@b.c", "subj", "body")
	fmt.Println(m)
}

func TestBase64Mail(t *testing.T) {
	m := newEmail("from@b.c", "to@b.c", "subj", "body")
	//fmt.Println(m.String())
	if s := m.base64URL(); len(s) == 0 {
		t.Errorf("unexpected value: %s", s)
	}
}

func TestEmailWithAttachmentString(t *testing.T) {
	m := newEmail("from@b.c", "to@b.c", "subj", "body", "test.csv", "test.csv")
	if s := m.String(); !strings.Contains(s, "test.csv") {
		t.Errorf("unexpected value: %s", s)
	}
	//fmt.Println("DEBUG test with attachment\n", m.String())
}
