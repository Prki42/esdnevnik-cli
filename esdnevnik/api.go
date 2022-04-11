package esdnevnik

import (
	"encoding/json"
	"math"
)

const (
	loginUrl = "https://moj.esdnevnik.rs/login"
	apiUrl   = "https://moj.esdnevnik.rs/api"
)

type Course struct {
	StudentClassCourseId int    `json:"studentClassCourseId"`
	ClassCourseId        int    `json:"classCourseId"`
	Name                 string `json:"name"`
	Sequence             int    `json:"sequence"`
}

type Class struct {
	Section        string         `json:"section"`
	StudentClassId int            `json:"studentClassId"`
	Courses        map[int]Course `json:"courses"`
}

type SchoolYear struct {
	Year    string        `json:"year"`
	YearId  int           `json:"year_id"`
	Classes map[int]Class `json:"classes"`
}

type School struct {
	SchoolName  string             `json:"schoolName"`
	SchoolYears map[int]SchoolYear `json:"schoolyears"`
}

type Student struct {
	Id       int            `json:"id"`
	FullName string         `json:"fullname"`
	Jmbg     string         `json:"jmbg"`
	Gender   string         `json:"gender"`
	Schools  map[int]School `json:"schools"`
}

type GetStudentsResponse struct {
	Data []Student `json:"data"`
}

type Grade struct {
	Descriptive      bool   `json:"descriptive"`
	Date             string `json:"date"`
	CreateDateTime   string `json:"createDatetime"`
	FullGrade        string `json:"fullGrade"`
	Grade            int    `json:"grade"`
	GradeCategory    string `json:"gradeCategory"`
	Note             string `json:"note"`
	SchoolYearPartId int    `json:"schoolYearPartId"`
}

type FinalGrade struct {
	Name             string `json:"name"`
	Value            int    `json:"value"`
	SchoolYearPartId int    `json:"schoolYearPartId"`
}

type Semester struct {
	Grades  []Grade     `json:"grades"`
	Final   FinalGrade  `json:"final"`
	Average json.Number `json:"average"`
}

type CourseGrading struct {
	Course                 string           `json:"course"`
	ClassCourseId          int              `json:"classCourseId"`
	ClassCourseGradeTypeId int              `json:"classCourseGradeTypeId"`
	Sequence               int              `json:"sequence"`
	Semesters              map[int]Semester `json:"parts"`
}

func (cg CourseGrading) getAverage() float64 {
	sum := 0
	count := 0
	for k := range cg.Semesters {
		for _, grade := range cg.Semesters[k].Grades {
			sum += grade.Grade
			count++
		}
	}
	return float64(sum) / float64(count)
}

func (cg CourseGrading) getFinal() int {
	if final, ok := cg.Semesters[2]; ok && final.Final.Value != 0 {
		return final.Final.Value
	}
	avg := cg.getAverage()
	if math.IsNaN(avg) || avg == 0 {
		return 0
	}
	if avg >= 4.5 {
		return 5
	}
	if avg >= 3.5 {
		return 4
	}
	if avg >= 2.5 {
		return 3
	}
	if avg >= 1.5 {
		return 2
	}
	return 1

}

type GetGradesResponse []CourseGrading
