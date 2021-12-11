package codegenerator

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

//var inlineTemplateRegex = regexp.MustCompile(`inline template: data file:`)
var inlineTemplateRegex = regexp.MustCompile(`\/\/ inline template: data file: "(.*)" *\r?\n((.|[\r\n])*?)\/\/ inline template end *\r?\n`)
var fileTemplateRegex = regexp.MustCompile(`\/\/ file template: data file: "(.*)" template file: "(.*)" *\r?\n`)

func ScanFolder(rootFolder string, extension string) error {
	return filepath.WalkDir(rootFolder, func(filename string, d os.DirEntry, err error) error {
		base := filepath.Base(filename)
		if base == "node_modules" || base == ".git" {
			return filepath.SkipDir
		}
		if strings.HasSuffix(filename, extension) {
			target := filename[:len(filename)-len(extension)]
			log.Println("Generating ", target)
			return SingleFile(filename, target)
		}
		return nil
	})
}

func SingleFile(inFilename string, outFilename string) error {
	contentsBytes, err := ioutil.ReadFile(inFilename)
	if err != nil {
		return err
	}
	for {
		match := inlineTemplateRegex.FindSubmatchIndex(contentsBytes)
		if len(match) == 0 {
			break
		}
		dataFilename := string(contentsBytes[match[2]:match[3]])
		templateContents := string(contentsBytes[match[4]:match[5]])
		parsedTemplate, err := template.New(fmt.Sprintf("%s:%d", inFilename, match[0])).Funcs(AuxilirayFunctions()).Parse(templateContents)
		if err != nil {
			log.Printf("While attempting to parse %s at position %d, template contents %s", inFilename, match[0], templateContents)
			return err
		}
		dataContents, err := ioutil.ReadFile(dataFilename)
		if err != nil {
			return err
		}
		var data interface{}
		err = yaml.Unmarshal(dataContents, &data)		
		if err != nil {
			return err
		}
		outputBuffer := bytes.Buffer{}
		outputWriter := bufio.NewWriter(&outputBuffer)
		err = parsedTemplate.Execute(outputWriter, data)
		if err != nil {
			return err
		}
		outputWriter.Flush()
		contentsBytes = append(append(contentsBytes[:match[0]], outputBuffer.Bytes()...), contentsBytes[match[1]:]...)
	}
	return ioutil.WriteFile(outFilename, contentsBytes, 0644)
}