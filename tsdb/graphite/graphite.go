package graphite

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	//"regexp"
	"strconv"
	"strings"
)

func joinUrlFragments(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

var GraphiteUrl *url.URL

func Init(graphiteUrl string) error {
	var err error
	GraphiteUrl, err = url.Parse(graphiteUrl)
	return err
}

func Proxy(owner int64, proxyPath string, request *http.Request) *httputil.ReverseProxy {
	/*
		// check if this is a special raintank_db requests
		if proxyPath == "metrics/find" {
			query := c.Query("query")
			if strings.HasPrefix(query, "raintank_db") {
				response, err := executeRaintankDbQuery(query, c.Owner)
				if err != nil {
					c.JsonApiErr(500, "Failed to execute raintank_db query", err)
					return
				}
				c.JSON(200, response)
				return
			}
		}
	*/

	director := func(req *http.Request) {
		req.URL.Scheme = GraphiteUrl.Scheme
		req.URL.Host = GraphiteUrl.Host
		req.Header.Add("X-Org-Id", strconv.FormatInt(owner, 10))
		req.URL.Path = joinUrlFragments(GraphiteUrl.Path, proxyPath)
	}

	return &httputil.ReverseProxy{Director: director}
}

/*
func executeRaintankDbQuery(query string, orgId int64) (interface{}, error) {
	values := []map[string]interface{}{}

	regex := regexp.MustCompile(`^raintank_db\.tags\.(\w+)\.(\w+|\*)`)
	matches := regex.FindAllStringSubmatch(query, -1)

	if len(matches) == 0 {
		return values, nil
	}

	tagType := matches[0][1]
	tagValue := matches[0][2]

	if tagType == "collectors" {
		if tagValue == "*" {
			// return all tags
			tagsQuery := m.GetAllCollectorTagsQuery{OrgId: orgId}
			if err := bus.Dispatch(&tagsQuery); err != nil {
				return nil, err
			}

			for _, tag := range tagsQuery.Result {
				values = append(values, util.DynMap{"text": tag, "expandable": false})
			}
			return values, nil
		} else if tagValue != "" {
			// return tag values for key
			collectorsQuery := m.GetCollectorsQuery{OrgId: orgId, Tag: []string{tagValue}}
			if err := bus.Dispatch(&collectorsQuery); err != nil {
				return nil, err
			}

			for _, collector := range collectorsQuery.Result {
				values = append(values, util.DynMap{"text": collector.Slug, "expandable": false})
			}
		}
	} else if tagType == "endpoints" {
		if tagValue == "*" {
			// return all tags
			tagsQuery := m.GetAllEndpointTagsQuery{OrgId: orgId}
			if err := bus.Dispatch(&tagsQuery); err != nil {
				return nil, err
			}

			for _, tag := range tagsQuery.Result {
				values = append(values, util.DynMap{"text": tag, "expandable": false})
			}
			return values, nil
		} else if tagValue != "" {
			// return tag values for key
			endpointsQuery := m.GetEndpointsQuery{OrgId: orgId, Tag: []string{tagValue}}
			if err := bus.Dispatch(&endpointsQuery); err != nil {
				return nil, err
			}

			for _, endpoint := range endpointsQuery.Result {
				values = append(values, util.DynMap{"text": endpoint.Slug, "expandable": false})
			}

		}
	}

	return values, nil
}
*/
