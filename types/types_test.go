// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphQlResponseErrorError(t *testing.T) {
	err := &GraphQlResponseError{Message: "user's role doesn't permit this action"}
	assert.EqualError(t, err, "user's role doesn't permit this action")
}

func TestGraphQlResponseErrorString(t *testing.T) {
	tests := []struct {
		name string
		err  GraphQlResponseError
		want string
	}{
		{
			name: "message only",
			err:  GraphQlResponseError{Message: "something went wrong"},
			want: "something went wrong",
		},
		{
			name: "path and error class",
			err: GraphQlResponseError{
				Message:    "user's role doesn't permit this action",
				Path:       []any{"actor", "account", "workload"},
				Extensions: &GraphQlResponseErrorExtensions{ErrorClass: "UNAUTHORIZED"},
			},
			want: "actor.account.workload: user's role doesn't permit this action (UNAUTHORIZED)",
		},
		{
			name: "list index in the path",
			err: GraphQlResponseError{
				Message: "not found",
				Path:    []any{"actor", "entities", 0, "tags"},
			},
			want: "actor.entities.0.tags: not found",
		},
		{
			name: "extensions without an error class",
			err: GraphQlResponseError{
				Message:    "boom",
				Path:       []any{"actor"},
				Extensions: &GraphQlResponseErrorExtensions{Code: "500"},
			},
			want: "actor: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.String())
		})
	}
}
