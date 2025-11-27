// Package govalid
// Author: Perry He
// Created on: 2025-11-27 08:35:02
package govalid

import "testing"

func TestValid(t *testing.T) {
	type Custom struct {
		Address string `json:"address" binding:"max=10,omitempty"`
		Hobby   string `json:"hobby" binding:"min=1,max=20"` // "omitempty" breaks the min=1 check
	}
	type Req struct {
		ID      int     `json:"id" binding:"required,min=1"`
		Age     int     `json:"age,omitempty" binding:"min=18"`
		Score   int     `json:"score" binding:"required,gte=60"`
		Custom  Custom  `json:"custom" binding:"required"` // must add `binding:"required"` for nested (嵌套) structs, otherwise (否则) it won't be validated
		CustomP *Custom `json:"custom_p" binding:"required"`
	}

	param := Req{
		ID:    1,
		Age:   18,
		Score: 99,
		Custom: Custom{
			Address: "",
			Hobby:   "coding",
		},
		CustomP: &Custom{
			Hobby: "coding",
		},
	}

	err := Valid(param)
	if err != nil {
		t.Fatal(err)
	}
}
