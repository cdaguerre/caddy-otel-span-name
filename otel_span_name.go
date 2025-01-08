package otelSpanName

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(UpdateSpanName{})
	httpcaddyfile.RegisterHandlerDirective("update_span_name", parseCaddyfileHandlerDirective)
	httpcaddyfile.RegisterDirectiveOrder("update_span_name", httpcaddyfile.After, "header")	
}

type UpdateSpanName struct {
	HeaderName       string `json:"header"`
	logger *zap.Logger
}

func (UpdateSpanName) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.otel_update_span_name",
		New: func() caddy.Module { return new(UpdateSpanName) },
	}
}

// Provision implements caddy.Provisioner.
func (usn *UpdateSpanName) Provision(ctx caddy.Context) error {	
	usn.logger = ctx.Logger()
	return nil
}

// Validate implements caddy.Validator.
func (usn *UpdateSpanName) Validate() error {
	return nil
}

// serveHTTP changes span name according to response header
func (usn *UpdateSpanName) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Execute the next handler and capture any error
	err := next.ServeHTTP(w, r)
	if err != nil {
		return err
	}

	spanCtx := trace.SpanContextFromContext(r.Context())

	if spanCtx.IsValid() {
		name := w.Header().Get(usn.HeaderName)

		usn.logger.Debug("Setting span name to " + name)

		if name != "" {
			span := trace.SpanFromContext(r.Context())
			span.SetName(name)

			cacheStatus := w.Header().Get("Cache-Status")

			if cacheStatus != "" {
				regexp, _ := regexp.Compile("^Souin; hit;(.*)")
				cacheHit := regexp.Match([]byte(cacheStatus))
				span.SetAttributes(attribute.KeyValue{Key: "cache.hit", Value: attribute.BoolValue(cacheHit) })
			}

		}
	} else {
		usn.logger.Debug("Span context invalid")
	}

	return nil
}

func (usn *UpdateSpanName) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	usn.HeaderName = "x-span-name"

	for d.Next() {
		if !d.NextArg() {
			return d.ArgErr()
		}
		if value := strings.TrimSpace(d.Val()); value != "" {
			usn.HeaderName = value
		}		
	}

	return nil
}

func parseCaddyfileHandlerDirective(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var usn UpdateSpanName
	return &usn, usn.UnmarshalCaddyfile(h.Dispenser)
}


// Interface guards
var (
	_ caddy.Provisioner           = (*UpdateSpanName)(nil)
	_ caddy.Validator             = (*UpdateSpanName)(nil)
	_ caddyhttp.MiddlewareHandler = (*UpdateSpanName)(nil)
	_ caddyfile.Unmarshaler       = (*UpdateSpanName)(nil)
)
