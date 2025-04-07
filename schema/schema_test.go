// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

import (
	"reflect"
	"testing"
)

func TestParseComment(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    []Section
	}{
		{
			name:    "test0",
			comment: `## ;;; @title "Architecture" @x-enum single="Single";cluster="Cluster Mode"`,
			want: []Section{
				{
					Name:  "@title",
					Value: "Architecture",
				},
				{
					Name: "@x-enum",
					Options: []Option{
						{Name: "single", Value: "Single"},
						{Name: "cluster", Value: "Cluster Mode"},
					},
				},
			},
		},
		{
			name:    "test1",
			comment: `# @enum 8.0.30`,
			want: []Section{
				{
					Name:  "@enum",
					Value: "8.0.30",
				},
			},
		},
		{
			name:    "test2",
			comment: `# @description PITR(Point-in-Time Recovery)`,
			want: []Section{
				{
					Name:  "@description",
					Value: "PITR(Point-in-Time Recovery)",
				},
			},
		},
		{
			name:    "empty",
			comment: `# @empty`,
			want: []Section{
				{
					Name: "@empty",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseComment(tt.comment)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseComment() got = %v, want %v", got, tt.want)
			}
		})
	}
}
