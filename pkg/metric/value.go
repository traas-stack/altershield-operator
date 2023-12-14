/*
 Copyright 2023 The Kapacity Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package metric

import (
	"time"

	prommodel "github.com/prometheus/common/model"
)

// Series is a stream of data points belonging to a metric.
type Series struct {
	Points []Point `json:"points"`
	Labels prommodel.LabelSet `json:"labels"`
	Window *time.Duration `json:"window"`
}

// Sample is a single sample belonging to a metric.
type Sample struct {
	Point `json:"point"`
	Labels prommodel.LabelSet `json:"labels"`
	Window *time.Duration `json:"window"`
}

// Point represents a single data point for a given timestamp.
type Point struct {
	Timestamp prommodel.Time `json:"timestamp"`
	Value     float64 `json:"value"`
}
