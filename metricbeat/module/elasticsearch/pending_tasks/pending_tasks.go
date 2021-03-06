// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pending_tasks

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "pending_tasks", New,
		mb.WithHostParser(elasticsearch.HostParser),
		mb.DefaultMetricSet(),
		mb.WithNamespace("elasticsearch.pending_tasks"),
	)
}

const (
	pendingTasksPath = "/_cluster/pending_tasks"
)

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*elasticsearch.MetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("the " + base.FullyQualifiedName() + " metricset is beta")

	ms, err := elasticsearch.NewMetricSet(base, pendingTasksPath)
	if err != nil {
		return nil, err
	}

	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	isMaster, err := elasticsearch.IsMaster(m.HTTP, m.HostData().SanitizedURI+pendingTasksPath)
	if err != nil {
		err := errors.Wrap(err, "error determining if connected Elasticsearch node is master")
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	// Not master, no event sent
	if !isMaster {
		m.Log.Debug("trying to fetch pending tasks from a non-master node")
		return
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	err = eventsMapping(r, content)
	if err != nil {
		m.Log.Error(err)
		return
	}
}
