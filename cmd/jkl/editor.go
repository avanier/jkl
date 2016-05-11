package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"reflect"

	"bufio"

	"otremblay.com/jkl"
)

// def get_editor do
// 	[System.get_env("EDITOR"), "nano", "vim", "vi"]
// 	|> Enum.find(nil, fn (ed) -> System.find_executable(ed) != nil end)
//   end
var editors = []string{os.Getenv("EDITOR"), "nano", "vim", "vi"}

// GetEditor returns the path to an editor, taking $EDITOR in account
func GetEditor() string {
	for _, ed := range editors {
		if p, err := exec.LookPath(ed); err == nil {
			return p
		}
	}
	log.Fatal("No editor available; use flags.")
	return ""
}

func copyInitial(dst io.WriteSeeker, initial io.Reader) {
	io.Copy(dst, initial)
	dst.Seek(0, 0)
}

func GetIssueFromTmpFile(initial io.Reader) (*jkl.Issue, error) {
	f, err := ioutil.TempFile(os.TempDir(), "jkl")
	if err != nil {
		return nil, err
	}
	copyInitial(f, initial)
	f2, err := GetTextFromFile(f)
	if err != nil {
		return nil, err
	}
	return IssueFromFile(f2), nil
}

func GetTextFromTmpFile(initial io.Reader) (io.Reader, error) {
	f, err := ioutil.TempFile(os.TempDir(), "jkl")
	if err != nil {
		return nil, err
	}
	copyInitial(f, initial)
	return GetTextFromFile(f)
}

func GetTextFromSpecifiedFile(filename string, initial io.Reader) (io.Reader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	if fi, err := f.Stat(); err == nil && fi.Size() == 0 {
		copyInitial(f, initial)
	}
	return GetTextFromFile(f)
}

func GetTextFromFile(file *os.File) (io.Reader, error) {
	cmd := exec.Command(GetEditor(), file.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	_, err = file.Seek(0, 0)
	return file, err
}

func GetIssueFromFile(filename string, initial io.Reader) (*jkl.Issue, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	if fi, err := f.Stat(); err == nil && fi.Size() == 0 {
		copyInitial(f, initial)
	}
	f2, err := GetTextFromFile(f)
	if err != nil {
		return nil, err
	}
	return IssueFromFile(f2), nil
}

var spacex = regexp.MustCompile(`\s`)

func IssueFromFile(f io.Reader) *jkl.Issue {
	iss := &jkl.Issue{Fields: &jkl.Fields{}}
	riss := reflect.ValueOf(iss).Elem()
	fieldsField := riss.FieldByName("Fields").Elem()
	currentField := reflect.Value{}
	brd := bufio.NewReader(f)
	for {
		b, _, err := brd.ReadLine()
		if err != nil {
			break
		}
		parts := strings.Split(string(b), ":")
		potentialField := spacex.ReplaceAllString(parts[0], "")

		if newfield := fieldsField.FieldByName(potentialField); newfield.IsValid() {
			parts = parts[1:len(parts)]
			if potentialField == "IssueType" {
				iss.Fields.IssueType = &jkl.IssueType{}
				currentField = reflect.Value{}
				f2 := newfield.Elem()
				f3 := f2.FieldByName("Name")
				f3.SetString(strings.TrimSpace(strings.Join(parts, ":")))
			} else if potentialField == "Project" {
				iss.Fields.Project = &jkl.Project{}
				currentField = reflect.Value{}
				f2 := newfield.Elem()
				f3 := f2.FieldByName("Key")
				f3.SetString(strings.TrimSpace(strings.Join(parts, ":")))
			} else {
				currentField = newfield
			}
		}
		if currentField.IsValid() {
			currentField.SetString(strings.TrimSpace(currentField.String() + "\n" + strings.Join(parts, ":")))
		}
	}
	return iss
}

func IssueFromList(list []string) *jkl.Issue {
	return IssueFromFile(bytes.NewBufferString(strings.Join(list, "\n")))
}
