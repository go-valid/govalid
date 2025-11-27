// Package govalid
// Author: Perry He
// Created on: 2025-11-27 08:35:02
package govalid

import (
	"fmt"
	"testing"
	"time"
)

func TestValid(t *testing.T) {
	type Custom struct {
		Address string `json:"address" binding:"min=1,max=10,omitempty"`
		Hobby   string `json:"hobby" binding:"min=1,max=20"` // "omitempty" breaks the min=1 check
	}
	type Req struct {
		ID        int       `json:"id" binding:"required,min=1"`
		Age       int       `json:"age" binding:"min=18"`
		Score     int       `json:"score" binding:"required,gte=60"`
		Custom    Custom    `json:"custom" binding:"required"` // must add `binding:"required"` for nested (嵌套) structs, otherwise (否则) it won't be validated
		CustomP   *Custom   `json:"custom_p" binding:"required"`
		StartTime time.Time `json:"start_time" binding:"min=2025-11-27T00:00:00Z"`
		EndTime   time.Time `json:"end_time" binding:"required"`
	}

	CNLoc, _ := time.LoadLocation("Asia/Shanghai")
	time.Local = CNLoc

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
		StartTime: time.Date(2025, 11, 27, 21, 11, 59, 0, time.UTC), // This should fail min validation
		EndTime:   time.Date(2025, 11, 27, 21, 11, 59, 0, time.UTC), // This should fail max validation
	}

	err := Valid(param)
	if err != nil {
		fmt.Println("Validation failed:", err)
	} else {
		fmt.Println("Validation passed!")
	}
}
