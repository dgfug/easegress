/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bridge

import (
	"net/http"

	"github.com/megaease/easegress/pkg/context"
	"github.com/megaease/easegress/pkg/filters"
	"github.com/megaease/easegress/pkg/logger"
	"github.com/megaease/easegress/pkg/protocols/httpprot"
)

const (
	// Kind is the kind of Bridge.
	Kind = "Bridge"

	// Description is the Description of Bridge.
	Description = `# Bridge Filter

A Bridge Filter route requests to from one pipeline to other pipelines or http proxies under a http server.

1. The upstream filter set the target pipeline/proxy to the http header,  'X-Easegress-Bridge-Dest'.
2. Bridge will extract the value from 'X-Easegress-Bridge-Dest' and try to match in the configuration.
   It will send the request if a dest matched. abort the process if no match.
3. Bridge will select the first dest from the filter configuration if there's no header named 'X-Easegress-Bridge-Dest'`

	resultDestinationNotFound     = "destinationNotFound"
	resultInvokeDestinationFailed = "invokeDestinationFailed"

	bridgeDestHeader = "X-Easegress-Bridge-Dest"
)

var kind = &filters.Kind{
	Name:        Kind,
	Description: Description,
	Results:     []string{resultDestinationNotFound, resultInvokeDestinationFailed},
	DefaultSpec: func() filters.Spec {
		return &Spec{}
	},
	CreateInstance: func() filters.Filter {
		return &Bridge{}
	},
}

func init() {
	// FIXME: Bridge is a temporary product for some historical reason.
	// I(@xxx7xxxx) think we should not empower filter to cross pipelines.

	// filters.Register(kind)
}

type (
	// Bridge is filter Bridge.
	Bridge struct {
		spec      *Spec
		muxMapper context.MuxMapper
	}

	// Spec describes the Mock.
	Spec struct {
		filters.BaseSpec `yaml:",inline"`
		Destinations     []string `yaml:"destinations" jsonschema:"required,pattern=^[^ \t]+$"`
	}
)

// Name returns the name of the Bridge filter instance.
func (b *Bridge) Name() string {
	return b.spec.Name()
}

// Kind returns the kind of Bridge.
func (b *Bridge) Kind() *filters.Kind {
	return kind
}

// Spec returns the spec used by the Bridge
func (b *Bridge) Spec() filters.Spec {
	return b.spec
}

// Init initializes Bridge.
func (b *Bridge) Init(spec filters.Spec) {
	b.spec = spec.(*Spec)
	b.reload()
}

// Inherit inherits previous generation of Bridge.
func (b *Bridge) Inherit(spec filters.Spec, previousGeneration filters.Filter) {
	previousGeneration.Close()
	b.Init(spec)
}

func (b *Bridge) reload() {
	if len(b.spec.Destinations) <= 0 {
		logger.Errorf("not any destination defined")
	}
}

// InjectMuxMapper injects mux mapper into Bridge.
func (b *Bridge) InjectMuxMapper(mapper context.MuxMapper) {
	b.muxMapper = mapper
}

// Handle builds a bridge for pipeline.
func (b *Bridge) Handle(ctx context.Context) (result string) {
	httpresp := ctx.Response().(*httpprot.Response)
	if len(b.spec.Destinations) <= 0 {
		panic("not any destination defined")
	}

	r := ctx.Request()
	dest := r.Header().Get(bridgeDestHeader)
	found := false
	if dest == "" {
		logger.Warnf("destination not defined, will choose the first dest: %s", b.spec.Destinations[0])
		dest = b.spec.Destinations[0]
		found = true
	} else {
		for _, d := range b.spec.Destinations {
			if d == dest {
				r.Header().Del(bridgeDestHeader)
				found = true
				break
			}
		}
	}

	if !found {
		logger.Errorf("dest not found: %s", dest)
		httpresp.SetStatusCode(http.StatusServiceUnavailable)
		return resultDestinationNotFound
	}

	handler, exists := b.muxMapper.GetHandler(dest)

	if !exists {
		logger.Errorf("failed to get running object %s", b.spec.Destinations[0])
		httpresp.SetStatusCode(http.StatusServiceUnavailable)
		return resultDestinationNotFound
	}

	handler.Handle(ctx)

	return ""
}

// Status returns status.
func (b *Bridge) Status() interface{} {
	return nil
}

// Close closes Bridge.
func (b *Bridge) Close() {}
