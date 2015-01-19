package septa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type floatFromString float64

func (f *floatFromString) UnmarshalJSON(b []byte) (err error) {
	var s string

	if err = json.Unmarshal(b, &s); err != nil {
		return
	}

	var g float64
	if g, err = strconv.ParseFloat(s, 64); err != nil {
		return
	}

	*f = floatFromString(g)

	return
}

type intFromString int

func (i *intFromString) UnmarshalJSON(b []byte) (err error) {
	var s string

	if err = json.Unmarshal(b, &s); err != nil {
		return
	}

	var j int
	if j, err = strconv.Atoi(s); err != nil {
		return
	}

	*i = intFromString(j)

	return
}

// BusTrolley represents the position of a bus or trolley.
type BusTrolley struct {
	Lat         float64
	Lng         float64
	LastRead    int
	Direction   string `json:"Direction"`
	Destination string `json:"destination"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (bt *BusTrolley) UnmarshalJSON(b []byte) (err error) {
	type busTrolleyJSON BusTrolley

	var btj struct {
		Lat      floatFromString `json:"lat"`
		Lng      floatFromString `json:"lng"`
		LastRead intFromString   `json:"Offset"`
		busTrolleyJSON
	}

	if err = json.Unmarshal(b, &btj); err != nil {
		return
	}

	*bt = BusTrolley(btj.busTrolleyJSON)
	bt.Lat = float64(btj.Lat)
	bt.Lng = float64(btj.Lng)
	bt.LastRead = int(btj.LastRead)

	return
}

func (bt BusTrolley) String() (s string) {
	base := `Latitude: %f
Longitude: %f
Direction: %s
Destination: %s
LastRead: %d minutes ago
`
	s = fmt.Sprintf(base, bt.Lat, bt.Lng, bt.Direction, bt.Destination,
		bt.LastRead)

	return
}

type busTrolleys []BusTrolley

func (bts *busTrolleys) UnmarshalJSON(b []byte) (err error) {

	var buses struct {
		Buses []BusTrolley `json:"bus"`
	}

	if err = json.Unmarshal(b, &buses); err != nil {
		return
	}

	*bts = buses.Buses

	return
}

// RouteAlert represents an alert on a SEPTA route.
type RouteAlert struct {
	RouteName       string `json:"route_name"`
	CurrentMessage  string `json:"current_message"`
	AdvisoryMessage string `json:"advisory_message"`
}

// HTTPClient implements SEPTA API functionality.
type HTTPClient struct {
	endpoint string
}

// TransitView returns the current transit status for the given route.
func (c HTTPClient) TransitView(route string) (bts []BusTrolley, err error) {
	url := fmt.Sprintf("%s/hackathon/TransitView/%s", c.endpoint, route)

	var resp *http.Response
	if resp, err = http.Get(url); err != nil {
		return
	}

	var ret []byte
	if ret, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}

	if err = resp.Body.Close(); err != nil {
		return
	}

	var tempBTS busTrolleys
	if err = json.Unmarshal(ret, &tempBTS); err != nil {
		return
	}

	bts = tempBTS

	return
}

func (c HTTPClient) traverseHTML(n *html.Node) (s string, err error) {
	b := bytes.Buffer{}
	var re *regexp.Regexp
	if re, err = regexp.Compile("\n\t* ?"); err != nil {
		return
	}

	addNewline := false
	switch n.Type {
	case html.TextNode:
		t := re.ReplaceAllString(n.Data, "")
		if t != "" {
			b.WriteString(t)
		}
	case html.ElementNode:
		if n.Data == "p" || n.Data == "h3" {
			addNewline = true
		}
	}

	for m := n.FirstChild; m != nil; m = m.NextSibling {
		var t string
		if t, err = c.traverseHTML(m); err != nil {
			return
		}
		b.WriteString(t)
	}

	if addNewline {
		b.WriteString("\n")
	}

	s = b.String()

	return
}

func (c HTTPClient) stripHTML(s string) (t string, err error) {
	var n *html.Node
	if n, err = html.Parse(strings.NewReader(s)); err != nil {
		return
	}

	t, err = c.traverseHTML(n)

	return
}

// RouteAlerts returns alerts for the given route.
func (c HTTPClient) RouteAlerts(route string) (rts []RouteAlert, err error) {
	url := fmt.Sprintf("%s/hackathon/Alerts/get_alert_data.php?req1=%s",
		c.endpoint, route)

	var resp *http.Response
	if resp, err = http.Get(url); err != nil {
		return
	}

	var ret []byte
	if ret, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}

	if err = resp.Body.Close(); err != nil {
		return
	}

	err = json.Unmarshal(ret, &rts)
	for i := range rts {
		rts[i].AdvisoryMessage, err = c.stripHTML(rts[i].AdvisoryMessage)
		if err != nil {
			return
		}
		rts[i].CurrentMessage, err = c.stripHTML(rts[i].CurrentMessage)
		if err != nil {
			return
		}
	}

	return
}
