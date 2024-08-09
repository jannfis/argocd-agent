// Copyright 2024 The argocd-agent Authors
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

package env

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Bool(t *testing.T) {
	t.Run("True value from env", func(t *testing.T) {
		t.Setenv("FOO", "true")
		b := BoolWithDefault("FOO", false)
		assert.True(t, b)
		t.Setenv("FOO", "TRUE")
		b = BoolWithDefault("FOO", false)
		assert.True(t, b)
	})
	t.Run("False value from env", func(t *testing.T) {
		t.Setenv("FOO", "false")
		b := BoolWithDefault("FOO", true)
		assert.False(t, b)
		t.Setenv("FOO", "FALSE")
		b = BoolWithDefault("FOO", true)
		assert.False(t, b)
	})
	t.Run("Default value from env", func(t *testing.T) {
		b := BoolWithDefault("FOO", true)
		assert.True(t, b)
		b = BoolWithDefault("FOO", false)
		assert.False(t, b)
		t.Setenv("FOO", "notabool")
		b = BoolWithDefault("FOO", true)
		assert.True(t, b)
	})
}

func Test_String(t *testing.T) {
	t.Run("Simple string from env", func(t *testing.T) {
		s := StringWithDefault("FOO", nil, "bar")
		assert.Equal(t, "bar", s)
		t.Setenv("FOO", "baz")
		s = StringWithDefault("FOO", nil, "bar")
		assert.Equal(t, "baz", s)
	})
	t.Run("Validated string from env", func(t *testing.T) {
		v := func(s string) error {
			if s != "baz" {
				return fmt.Errorf("string doesn't match baz")
			}
			return nil
		}
		t.Setenv("FOO", "bar")
		s := StringWithDefault("FOO", v, "baz")
		assert.Equal(t, "baz", s)
		t.Setenv("FOO", "baz")
		s = StringWithDefault("FOO", v, "bar")
		assert.Equal(t, "baz", s)
	})

}

func Test_Num(t *testing.T) {
	t.Run("Test numeric value from env", func(t *testing.T) {
		t.Setenv("FOO", "20")
		n, err := Num("FOO", nil)
		assert.NoError(t, err)
		assert.Equal(t, 20, n)
		t.Setenv("FOO", "-20")
		n, err = Num("FOO", nil)
		assert.NoError(t, err)
		assert.Equal(t, -20, n)
		_, err = Num("BAR", nil)
		assert.ErrorIs(t, err, os.ErrNotExist)
		n = NumWithDefault("BAR", nil, 20)
		assert.Equal(t, 20, n)
	})
	t.Run("Test validated numeric value from env", func(t *testing.T) {
		v := func(num int) error {
			if num < 0 || num > 20 {
				return fmt.Errorf("invalid")
			}
			return nil
		}
		t.Setenv("FOO", "20")
		n, err := Num("FOO", v)
		assert.NoError(t, err)
		assert.Equal(t, 20, n)
		n = NumWithDefault("FOO", v, 30)
		assert.Equal(t, 20, n)
		t.Setenv("FOO", "-20")
		_, err = Num("FOO", v)
		assert.ErrorContains(t, err, "invalid")
		n = NumWithDefault("FOO", v, 20)
		assert.Equal(t, 20, n)
	})
	t.Run("Non-numeric value from env", func(t *testing.T) {
		t.Setenv("FOO", "bar")
		_, err := Num("FOO", nil)
		assert.ErrorContains(t, err, "ParseInt")
		n := NumWithDefault("FOO", nil, 20)
		assert.Equal(t, 20, n)
	})
}

func Test_StringSlice(t *testing.T) {
	t.Run("Test valid string slice from env", func(t *testing.T) {
		t.Setenv("FOO", "foo")
		s, err := StringSlice("FOO", nil)
		assert.NoError(t, err)
		assert.Len(t, s, 1)
		assert.Equal(t, []string{"foo"}, s)
		t.Setenv("FOO", "foo, bar, baz")
		s, err = StringSlice("FOO", nil)
		assert.NoError(t, err)
		assert.Len(t, s, 3)
		assert.Equal(t, []string{"foo", "bar", "baz"}, s)
	})
	t.Run("Test valid string slice from env with validator", func(t *testing.T) {
		v := func(s string) error {
			if s != "foo" {
				return fmt.Errorf("invalid")
			}
			return nil
		}
		t.Setenv("FOO", "foo")
		s, err := StringSlice("FOO", v)
		assert.NoError(t, err)
		assert.Len(t, s, 1)
		assert.Equal(t, []string{"foo"}, s)
		t.Setenv("FOO", "foo, foo, foo")
		s, err = StringSlice("FOO", v)
		assert.NoError(t, err)
		assert.Len(t, s, 3)
		assert.Equal(t, []string{"foo", "foo", "foo"}, s)
		t.Setenv("FOO", "foo, bar, baz")
		_, err = StringSlice("FOO", v)
		assert.ErrorContains(t, err, "invalid")
	})
}
