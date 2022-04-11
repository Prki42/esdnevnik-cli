package esdnevnik

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chzyer/readline"
	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

type noBellStdout struct{}

func (n *noBellStdout) Write(p []byte) (int, error) {
	if len(p) == 1 && p[0] == readline.CharBell {
		return 0, nil
	}
	return readline.Stdout.Write(p)
}

func (n *noBellStdout) Close() error {
	return readline.Stdout.Close()
}

var NoBellStdout = &noBellStdout{}

func CLI(args []string) int {
	var app AppEnv
	err := app.fromArgs(args)
	if err != nil {
		return 2
	}
	if err = app.run(); err != nil {
		switch err.(type) {
		case RuntimeError:
			fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		}
	}
	return 0
}

type AppEnv struct {
	httpClient *http.Client
	username   string
	password   string
	classId    string
}

type RuntimeError struct {
	Err error
}

func (e RuntimeError) Error() string {
	return e.Err.Error()
}

func (app *AppEnv) fromArgs(args []string) error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	fl := flag.NewFlagSet("esdnevnik-cli", flag.ContinueOnError)
	fl.StringVar(
		&app.username, "u", "", "E-mail",
	)
	if err := fl.Parse(args); err != nil {
		return RuntimeError{Err: err}
	}
	if app.username == "" {
		fmt.Println("User is required")
		return errors.New("user not specified")
	}

	app.httpClient = client
	return nil
}

func (app *AppEnv) login() error {
	resp, err := app.httpClient.Get("https://moj.esdnevnik.rs/login")
	if err != nil {
		fmt.Println("login error")
		return err
	}
	defer resp.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	tokenSelection := doc.Find("form input[name=_token]").First()
	if tokenSelection == nil {
		fmt.Println("token input not found")
		return RuntimeError{Err: err}
	}
	token, hasToken := tokenSelection.Attr("value")
	if !hasToken {
		return RuntimeError{Err: errors.New("token has no value")}
	}

	formData := url.Values{}
	formData.Set("_token", token)
	formData.Set("username", app.username)
	formData.Set("password", app.password)
	r, err := http.NewRequest("POST", loginUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		fmt.Println("Error creating login post request")
		return RuntimeError{Err: err}
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(formData.Encode())))

	resp, err = app.httpClient.Do(r)
	if err != nil {
		fmt.Println("Error sending login POST request")
		return RuntimeError{Err: err}
	}

	if strings.HasSuffix(resp.Header.Get("Location"), "login") {
		fmt.Println("Login failed")
		return errors.New("login fail")
	}

	return nil
}

func (app *AppEnv) fetchStudents() (GetStudentsResponse, error) {
	r, err := http.NewRequest("GET", apiUrl+"/students", nil)
	if err != nil {
		fmt.Println("Error creating student fetch request")
		return GetStudentsResponse{}, RuntimeError{Err: err}
	}
	resp, err := app.httpClient.Do(r)
	if err != nil {
		fmt.Println("Error fetching students")
		return GetStudentsResponse{}, RuntimeError{Err: err}
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var respJson GetStudentsResponse
	err = decoder.Decode(&respJson)

	if err != nil {
		fmt.Println("Error parsing student data")
		return GetStudentsResponse{}, RuntimeError{Err: err}
	}

	return respJson, nil
}

func (app *AppEnv) fetchGrades() (GetGradesResponse, error) {
	r, err := http.NewRequest("GET", apiUrl+"/grades/"+app.classId, nil)
	if err != nil {
		fmt.Println("Error creating grades fetch request")
		return GetGradesResponse{}, RuntimeError{Err: err}
	}
	resp, err := app.httpClient.Do(r)
	if err != nil {
		fmt.Println("Error fetching grades")
		return GetGradesResponse{}, RuntimeError{Err: err}
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var respJson GetGradesResponse
	err = decoder.Decode(&respJson)

	if err != nil {
		fmt.Println("Error parsing grades data")
		return GetGradesResponse{}, RuntimeError{Err: err}
	}

	return respJson, nil
}

func (app *AppEnv) getPassword() error {
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return RuntimeError{Err: err}
	}
	app.password = string(password)

	return nil
}

func (app *AppEnv) run() error {
	fmt.Printf("Nalog: %v\n\n", app.username)

	fmt.Printf("Password: ")
	app.getPassword()
	fmt.Printf("\r \r")

	err := app.login()
	if err != nil {
		return err
	}
	students, err := app.fetchStudents()
	if err != nil {
		return err
	}

	var student Student

	if len(students.Data) == 0 {
		fmt.Println("No students found")
		return errors.New("no students")
	}

	if len(students.Data) == 1 {
		student = students.Data[0]
	} else {
		template := &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "> {{ .FullName | bold }}",
			Inactive: "{{ .FullName }}",
			Details:  "----------Ucenik----------\nIme i prezime: {{ .FullName }}\nPol: {{ .Gender }}",
		}
		prompt := promptui.Select{
			Label:        "Izaberi ucenika",
			Items:        students.Data,
			Templates:    template,
			Size:         4,
			HideSelected: true,
			Stdout:       NoBellStdout,
		}
		i, _, err := prompt.Run()
		if err != nil {
			return RuntimeError{Err: err}
		}
		student = students.Data[i]
	}
	fmt.Printf("Izabran ucenik: %v\n", student.FullName)

	var school School
	schoolNumber := 0
	lastSchoolId := 0
	for k := range student.Schools {
		schoolNumber++
		lastSchoolId = k
	}
	if schoolNumber == 0 {
		fmt.Println("No schools found for selected ucenik")
		return errors.New("no schools")
	}
	if schoolNumber == 1 {
		school = student.Schools[lastSchoolId]
	} else {
		fmt.Println("Multiple schools not supported yet")
		return errors.New("multiple schools")
	}
	fmt.Printf("Skola: %v\n", school.SchoolName)

	var schoolYear SchoolYear
	schoolYears := make([]SchoolYear, 0, len(school.SchoolYears))
	for _, v := range school.SchoolYears {
		schoolYears = append(schoolYears, v)
	}
	if len(schoolYears) == 0 {
		fmt.Println("No school years found")
		return errors.New("no school years")
	}
	if len(schoolYears) == 1 {
		schoolYear = schoolYears[0]
	} else {
		template := &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "> {{ .Year | bold }}",
			Inactive: "{{ .Year }}",
		}
		prompt := promptui.Select{
			Label:        "Izaberi godinu",
			Items:        schoolYears,
			Templates:    template,
			Size:         4,
			HideSelected: true,
			Stdout:       NoBellStdout,
		}
		i, _, err := prompt.Run()
		if err != nil {
			return RuntimeError{Err: err}
		}
		schoolYear = schoolYears[i]
	}
	fmt.Printf("Izabrana godina: %v\n", schoolYear.Year)

	if len(schoolYear.Classes) == 0 {
		fmt.Println("No classes found")
		return errors.New("no classes")
	}
	if len(schoolYear.Classes) != 1 {
		fmt.Println("Schoolyears with multiplce classes not supported")
		return errors.New("multiple classes")
	}
	var classId int
	for k := range schoolYear.Classes {
		classId = k
	}
	class := schoolYear.Classes[classId]
	fmt.Printf("Odeljenje: %v\n", class.Section)
	app.classId = fmt.Sprintf("%v", class.StudentClassId)

	prompt := promptui.Select{
		Label:        "Vladanje",
		Items:        []int{1, 2, 3, 4, 5},
		HideSelected: true,
		Stdout:       NoBellStdout,
	}
	i, _, err := prompt.Run()
	if err != nil {
		return RuntimeError{Err: err}
	}
	vladanje := i + 1

	fmt.Println()

	grades, err := app.fetchGrades()
	if err != nil {
		return err
	}

	sum := vladanje
	count := 1

	fmt.Printf("Vladanje: %v\n", vladanje)
	for _, c := range grades {
		zakljuceno := c.getFinal()
		if zakljuceno == 0 {
			continue
		}
		sum += zakljuceno
		count++
		fmt.Printf("%v: %.2f %d\n", c.Course, c.getAverage(), zakljuceno)
	}
	fmt.Printf("\nProsek na kraju: %.2f\n", float64(sum)/float64(count))

	return nil
}
