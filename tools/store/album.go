/*
* Copyright © 2018-2019 private, Darmstadt, Germany and/or its licensors
*
* SPDX-License-Identifier: Apache-2.0
*
*   Licensed under the Apache License, Version 2.0 (the "License");
*   you may not use this file except in compliance with the License.
*   You may obtain a copy of the License at
*
*       http://www.apache.org/licenses/LICENSE-2.0
*
*   Unless required by applicable law or agreed to in writing, software
*   distributed under the License is distributed on an "AS IS" BASIS,
*   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*   See the License for the specific language governing permissions and
*   limitations under the License.
*
 */

package store

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
)

// Picture picture description
type Picture struct {
	Description string
	Name        string
	Md5         string
	Interval    uint32
	MIMEType    string
	Width       uint32
	Height      uint32
	Fill        string
}

// Album album information
type Album struct {
	path             string   `xml:"-" json:"-"`
	fileName         string   `xml:"-" json:"-"`
	file             *os.File `xml:"-" json:"-"`
	Directory        string
	Date             uint64
	Key              string
	Title            string
	AlbumDescription string
	Thumbnail        string
	Pictures         []*Picture
}

// AlbumName name of map for album
var AlbumName string

func createStringMd5(input string) string {
	h := md5.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%X", h.Sum(nil))
}
