package main

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/fabianvf/windup-rulesets-yaml/pkg/windup"
	"gopkg.in/yaml.v2"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("The location of your windup directory must be passed")
	}
	windupLocation := os.Args[1]
	rulesets := []windup.Ruleset{}
	err := filepath.WalkDir(windupLocation+"/rules/", processXML(windupLocation, &rulesets))
	if err != nil {
		fmt.Println(err)
	}
	err = convertWindupToAnalyzer(rulesets)
	if err != nil {
		log.Fatal(err)
	}
}

func convertWindupDependencyToAnalyzer(windupDependency windup.Dependency) map[string]interface{} {
	dependency := map[string]interface{}{
		"groupID":    windupDependency.GroupId,
		"artifactID": windupDependency.ArtifactId,
	}

	if windupDependency.FromVersion != "" {
		dependency["lowerBound"] = windupDependency.FromVersion
	}
	if windupDependency.ToVersion != "" {
		dependency["upperBound"] = windupDependency.ToVersion
	}
	return dependency
}

func convertWindupWhenToAnalyzer(windupWhen windup.When) []map[string]interface{} {
	// TODO Rule.When
	// TODO - Graphquery
	conditions := []map[string]interface{}{}
	if windupWhen.Project != nil {
		for _, pc := range windupWhen.Project {
			conditions = append(conditions, map[string]interface{}{"java.dependency": convertWindupDependencyToAnalyzer(pc.Artifact)})
		}
	}
	if windupWhen.Dependency != nil {
		for _, dc := range windupWhen.Dependency {
			conditions = append(conditions, map[string]interface{}{"java.dependency": convertWindupDependencyToAnalyzer(dc)})
		}
	}
	if windupWhen.And != nil {
		whens := []map[string]interface{}{}
		for _, condition := range windupWhen.And {
			converted := convertWindupWhenToAnalyzer(condition)
			for _, c := range converted {
				whens = append(whens, c)
			}
		}
		conditions = append(conditions, map[string]interface{}{"and": whens})
	}
	if windupWhen.Or != nil {
		whens := []map[string]interface{}{}
		for _, condition := range windupWhen.Or {
			converted := convertWindupWhenToAnalyzer(condition)
			for _, c := range converted {
				whens = append(whens, c)
			}
		}
		conditions = append(conditions, map[string]interface{}{"or": whens})
	}
	if windupWhen.Not != nil {
		for _, condition := range windupWhen.Not {
			converted := convertWindupWhenToAnalyzer(condition)
			for _, c := range converted {
				c["not"] = true
				conditions = append(conditions, c)
			}
		}
	}
	if windupWhen.Filecontent != nil {
		for _, fc := range windupWhen.Filecontent {
			condition := map[string]interface{}{
				"filecontent": map[string]interface{}{
					"pattern":  fc.Pattern,
					"filename": fc.Filename,
					// TODO Filecontent.Filename needs to be implemented in analyzer
				},
			}
			if fc.As != "" {
				condition["as"] = fc.As
			}
			if fc.From != "" {
				condition["from"] = fc.From
			}
			conditions = append(conditions, condition)
		}
	}
	if windupWhen.Javaclass != nil {
		for _, jc := range windupWhen.Javaclass {
			if jc.Location != nil {
				for _, location := range jc.Location {
					condition := map[string]interface{}{
						"java.referenced": map[string]interface{}{
							// TODO handle list of locations
							// TODO handle translation of windup locations to analyzer locations
							"location": location,
							"pattern":  jc.References,
							// TODO handle jc.Annotationtype
							// TODO handle jc.Annotationlist
							// TODO handle jc.Annotationliteral
							// TODO handle jc.MatchesSource
							// TODO handle jc.In
						},
					}
					if jc.As != "" {
						condition["as"] = jc.As
					}
					if jc.From != "" {
						condition["from"] = jc.From
					}
					conditions = append(conditions, condition)
				}
			} else {
				condition := map[string]interface{}{
					"java.referenced": map[string]interface{}{
						// TODO handle list of locations
						// TODO handle translation of windup locations to analyzer locations
						"pattern": jc.References,
						// TODO handle jc.Annotationtype
						// TODO handle jc.Annotationlist
						// TODO handle jc.Annotationliteral
						// TODO handle jc.MatchesSource
						// TODO handle jc.In
					},
				}
				if jc.As != "" {
					condition["as"] = jc.As
				}
				if jc.From != "" {
					condition["from"] = jc.From
				}
				conditions = append(conditions, condition)
			}
		}
	}
	if windupWhen.Xmlfile != nil {
		conditions = append(conditions, map[string]interface{}{"xmlfile": nil})
	}
	if windupWhen.File != nil {
		conditions = append(conditions, map[string]interface{}{"file": nil})
	}
	if windupWhen.Fileexists != nil {
		conditions = append(conditions, map[string]interface{}{"file-exists": nil})
	}
	if windupWhen.True != "" {
		conditions = append(conditions, map[string]interface{}{"true": nil})
	}
	if windupWhen.False != "" {
		conditions = append(conditions, map[string]interface{}{"false": nil})
	}
	if windupWhen.Classificationexists != nil {
		conditions = append(conditions, map[string]interface{}{"classification-exists": nil})
	}
	if windupWhen.Hintexists != nil {
		conditions = append(conditions, map[string]interface{}{"hint-exists": nil})
	}
	if windupWhen.Lineitemexists != nil {
		conditions = append(conditions, map[string]interface{}{"lineitem-exists": nil})
	}
	if windupWhen.Technologystatisticsexists != nil {
		conditions = append(conditions, map[string]interface{}{"technology-statistics-exists": nil})
	}
	if windupWhen.Technologytagexists != nil {
		conditions = append(conditions, map[string]interface{}{"technology-tag-exists": nil})
	}
	if windupWhen.Iterablefilter != nil {
		conditions = append(conditions, map[string]interface{}{"iterable-filter": nil})
	}
	if windupWhen.Tofilemodel != nil {
		conditions = append(conditions, map[string]interface{}{"tofilemodel": nil})
	}
	return conditions
}

func convertWindupToAnalyzer(windups []windup.Ruleset) error {
	outputRulesets := map[string][]map[string]interface{}{}
	for _, windupRuleset := range windups {
		// TODO Ruleset.Metadata
		// TODO Ruleset.PackageMapping
		// TODO Ruleset.Filemapping
		// TODO Ruleset.Javaclassignore
		ruleset := map[string]interface{}{}
		rules := []map[string]interface{}{}
		for i, windupRule := range windupRuleset.Rules.Rule {
			formatted, _ := yaml.Marshal(windupRule)
			fmt.Println(string(formatted))
			rule := map[string]interface{}{
				"ruleID": windupRuleset.SourceFile + "-" + strconv.Itoa(i),
			}
			if !reflect.DeepEqual(windupRule.When, windup.When{}) {
				when := convertWindupWhenToAnalyzer(windupRule.When)
				if len(when) == 0 {
					continue
				}
				if len(when) > 1 {
					rule["when"] = map[string]interface{}{"or": when}
				} else {
					rule["when"] = when
				}
			} else {
				continue
			}

			// TODO Rule.Perform
			// TODO - Iteration
			// TODO Rule.Otherwise
			// TODO Rule.Where
			rules = append(rules, rule)
		}
		ruleset["rules"] = rules
		yamlPath := strings.Replace(strings.Replace(windupRuleset.SourceFile, "http://github.com/windup/windup-rulesets/tree/master/", "analyzer-lsp-rules/", 1), ".xml", ".yaml", 1)
		if reflect.DeepEqual(ruleset, map[string]interface{}{}) {
			continue
		}
		outputRulesets[yamlPath] = append(outputRulesets[yamlPath], ruleset)
	}
	for path, ruleset := range outputRulesets {
		dirName := filepath.Dir(path)
		err := os.MkdirAll(dirName, 0777)
		if err != nil {
			fmt.Printf("Skipping because of an error creating %s: %s\n", path, err.Error())
			continue
		}
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Printf("Skipping because of an error opening %s: %s\n", path, err.Error())
			continue
		}
		defer file.Close()
		formatted, _ := yaml.Marshal(ruleset)
		fmt.Println(string(formatted))

		enc := yaml.NewEncoder(file)

		err = enc.Encode(ruleset)
		if err != nil {
			fmt.Printf("Skipping %s because of an error writing the yaml: %s\n", path, err.Error())
			continue
		}
	}
	return nil
}

func processXML(root string, rulesets *[]windup.Ruleset) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if !strings.HasSuffix(path, ".xml") {
			fmt.Printf("Skipping %s because it is not a ruleset\n", path)
			return nil
		}
		xmlFile, err := os.Open(path)
		if err != nil {
			fmt.Printf("Skipping %s because of an error opening the file: %s\n", path, err.Error())
			return nil
		}
		defer xmlFile.Close()
		byteValue, err := ioutil.ReadAll(xmlFile)
		if err != nil {
			fmt.Printf("Skipping %s because of an error reading the file: %s\n", path, err.Error())
			return nil
		}

		var ruleset windup.Ruleset

		err = xml.Unmarshal(byteValue, &ruleset)
		if err != nil {
			fmt.Printf("Skipping %s because of an error unmarhsaling the file: %s\n", path, err.Error())
			return nil
		}
		if reflect.ValueOf(ruleset).IsZero() {
			// TODO parse tests as well
			fmt.Printf("Skipping %s because it is not a ruleset\n", path)
			return nil
		}

		ruleset.SourceFile = strings.Replace(path, root, "http://github.com/windup/windup-rulesets/tree/master/", 1)
		*rulesets = append(*rulesets, ruleset)

		yamlPath := strings.Replace(strings.Replace(path, root, "converted/", 1), ".xml", ".yaml", 1)
		dirName := filepath.Dir(yamlPath)
		err = os.MkdirAll(dirName, 0777)
		if err != nil {
			fmt.Printf("Skipping %s because of an error creating %s: %s\n", path, yamlPath, err.Error())
			return nil
		}
		file, err := os.OpenFile(yamlPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Printf("Skipping %s because of an error opening %s: %s\n", path, yamlPath, err.Error())
			return nil
		}
		defer file.Close()

		enc := yaml.NewEncoder(file)

		err = enc.Encode(ruleset)
		if err != nil {
			fmt.Printf("Skipping %s because of an error writing the yaml: %s\n", path, err.Error())
			return nil
		}
		return nil
	}
	return nil
}