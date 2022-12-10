package glue

import (
	"encoding/json"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"testing"
)

var exampleScript = `
person = {
  first_name = "Jan",
  last_name = "Novak",
  age = "31", -- weak input (not int)
  workplace = "Prague",
  roles = {
	{
	  name = "Administrator"
	},
	{
	  name = "Operator"
	}
  }
}
`

var exampleScript1 = `
function dump(o)
   if type(o) == 'table' then
      local s = '{ '
      for k,v in pairs(o) do
         if type(k) ~= 'number' then k = '"'..k..'"' end
         s = s .. '['..k..'] = ' .. dump(v) .. ','
      end
      return s .. '} '
   else
      return tostring(o)
   end
end

print(dump(person))
`

type Role struct {
	Name string `json:"name" glue:"name"`
}

type Person struct {
	FirstName string  `json:"first_name" glue:"first_name"`
	LastName  string  `json:"last_name" glue:"last_name"`
	Age       int     `json:"age" glue:"age"`
	Workplace string  `json:"workplace" glue:"workplace"`
	Roles     []*Role `json:"roles" glue:"roles"`
}

type Role1 struct {
	Name string `json:"name"`
}

type Person1 struct {
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Age       int      `json:"age"`
	Workplace string   `json:"workplace"`
	Roles     []*Role1 `json:"roles"`
}

func TestMapper_Decode(t *testing.T) {
	l := lua.NewState()
	if err := l.DoString(exampleScript); err != nil {
		t.Fatalf("failed to interpret script: %s", err.Error())
	}

	m := NewMapper(0)

	var person Person
	if err := m.Decode(l.GetGlobal("person").(*lua.LTable), &person); err != nil {
		t.Fatalf("failed to decode: %s", err.Error())
	}

	bytes, err := json.MarshalIndent(person, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %s", err.Error())
	}

	fmt.Println(string(bytes))
}

func TestMapper_Encode(t *testing.T) {
	m := NewMapper(0)

	person := Person{
		FirstName: "Jan",
		LastName:  "Novak",
		Age:       31,
		Workplace: "Prague",
		Roles: []*Role{
			{
				Name: "Administrator",
			},
			{
				Name: "Operator",
			},
		},
	}

	var table lua.LTable
	if err := m.Encode(person, &table); err != nil {
		t.Fatalf("failed to encode: %s", err.Error())
	}

	l := lua.NewState()
	l.SetGlobal("person", &table)
	if err := l.DoString(exampleScript1); err != nil {
		t.Fatalf("failed to interpret script: %s", err.Error())
	}
}

func TestMapper_DecodeNoTags(t *testing.T) {
	l := lua.NewState()
	if err := l.DoString(exampleScript); err != nil {
		t.Fatalf("failed to interpret script: %s", err.Error())
	}

	m := NewMapper(OptionsSnakeCaseNaming)

	var person Person1
	if err := m.Decode(l.GetGlobal("person").(*lua.LTable), &person); err != nil {
		t.Fatalf("failed to decode: %s", err.Error())
	}

	bytes, err := json.MarshalIndent(person, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %s", err.Error())
	}

	fmt.Println(string(bytes))
}

func TestMapper_EncodeNoTags(t *testing.T) {
	m := NewMapper(OptionsSnakeCaseNaming)

	person := Person1{
		FirstName: "Jan",
		LastName:  "Novak",
		Age:       31,
		Workplace: "Prague",
		Roles: []*Role1{
			{
				Name: "Administrator",
			},
			{
				Name: "Operator",
			},
		},
	}

	var table lua.LTable
	if err := m.Encode(person, &table); err != nil {
		t.Fatalf("failed to encode: %s", err.Error())
	}

	l := lua.NewState()
	l.SetGlobal("person", &table)
	if err := l.DoString(exampleScript1); err != nil {
		t.Fatalf("failed to interpret script: %s", err.Error())
	}
}
