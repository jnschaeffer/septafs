// Package septa implements a simple read-only SEPTA file system, septafs.
package septa

import (
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var client = HTTPClient{
	endpoint: "http://www3.septa.org",
}

var tRouteIDs = []string{"10", "11", "13", "15", "34", "36", "101", "102"}
var bRouteIDs = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "12",
	"14", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25", "26", "27",
	"28", "29", "30", "31", "32", "33", "35", "37", "38", "39", "40", "42", "43",
	"44", "46", "47", "47m", "48", "50", "52", "53", "54", "55", "56", "57", "58",
	"59", "60", "61", "62", "64", "65", "66", "67", "68", "70", "73", "75", "77",
	"78", "79", "80", "84", "88", "89", "G", "H", "XH", "J", "K", "L", "R", "LUCY",
	"90", "91", "92", "93", "94", "95", "96", "97", "98", "99", "103", "104",
	"105", "106", "107", "108", "109", "110", "111", "112", "113", "114", "115",
	"116", "117", "118", "119", "120", "123", "124", "125", "126", "127", "128",
	"129", "130", "131", "132", "133", "139", "150", "201", "204", "205", "206",
	"310"}

// FS implements the SEPTA file system, septafs.
type FS struct{}

// Root returns a rootDir as the file system root.
func (FS) Root() (n fs.Node, err fuse.Error) {
	n = rootDir{
		trolleyNode: newBusTrolleyRoutes(tRouteIDs, false, 2),
		busNode:     newBusTrolleyRoutes(bRouteIDs, true, 3),
	}

	return
}

// rootDir implements Node and Handle for the SEPTA file system. At the root
// of septafs are the following directories:
type rootDir struct {
	trolleyNode busTrolleyRoutes
	busNode     busTrolleyRoutes
}

// Attr returns the rootDir attributes.
func (rootDir) Attr() fuse.Attr {
	return fuse.Attr{
		Inode: 1,
		Mode:  os.ModeDir | 0555,
	}
}

// ReadDir returns the contents of rootDir as detailed in the rootDir
// documentation.
func (rootDir) ReadDir(intr fs.Intr) (dirs []fuse.Dirent, err fuse.Error) {
	dirs = []fuse.Dirent{
		{Name: "trolley", Type: fuse.DT_Dir},
		{Name: "bus", Type: fuse.DT_Dir},
	}

	return
}

// Lookup implements node lookup in FUSE.
func (r rootDir) Lookup(name string, intr fs.Intr) (n fs.Node,
	err fuse.Error) {
	switch name {
	case "trolley":
		n = r.trolleyNode
	case "bus":
		n = r.busNode
	default:
		err = fuse.ENOENT
	}

	return
}

// busTrolleyRoutes represents a directory for all trolley routes.
type busTrolleyRoutes struct {
	routeNodes map[string]busTrolleyRoute
	routeIDs   []string
	inode      uint64
}

func newBusTrolleyRoutes(routes []string, isBus bool, inode uint64) (
	r busTrolleyRoutes) {
	r.routeNodes = make(map[string]busTrolleyRoute, len(routes))
	r.routeIDs = routes
	r.inode = inode

	for _, id := range routes {
		inode := fs.GenerateDynamicInode(r.inode, id)
		r.routeNodes[id] = newBusTrolleyRoute(id, inode, isBus)
	}

	return
}

func (r busTrolleyRoutes) Attr() fuse.Attr {
	return fuse.Attr{
		Inode: r.inode,
		Mode:  os.ModeDir | 055,
	}
}

func (r busTrolleyRoutes) Lookup(name string, intr fs.Intr) (n fs.Node,
	err fuse.Error) {

	var ok bool
	if n, ok = r.routeNodes[name]; !ok {
		err = fuse.ENOENT
		return
	}

	return
}

func (r busTrolleyRoutes) ReadDir(intr fs.Intr) (dirs []fuse.Dirent,
	err fuse.Error) {

	dirs = make([]fuse.Dirent, len(r.routeIDs))

	for i, id := range r.routeIDs {
		dirs[i] = fuse.Dirent{Name: id, Type: fuse.DT_Dir}
	}

	return
}

// busTrolleyRoute represents a directory for bus/trolley data.
type busTrolleyRoute struct {
	route        string
	inode        uint64
	isBus        bool
	locationNode busTrolleyLocation
	alertsNode   routeAlerts
}

func newBusTrolleyRoute(route string, inode uint64,
	isBus bool) (b busTrolleyRoute) {
	locations := busTrolleyLocation{
		route: route,
		inode: fs.GenerateDynamicInode(inode, "locations"),
	}

	var routeName string
	if isBus {
		routeName = fmt.Sprintf("bus_route_%s", route)
	} else {
		routeName = fmt.Sprintf("trolley_route_%s", route)
	}

	alerts := routeAlerts{
		route: routeName,
		inode: fs.GenerateDynamicInode(inode, "alerts"),
	}

	b.route = route
	b.inode = inode
	b.isBus = isBus
	b.locationNode = locations
	b.alertsNode = alerts

	return
}

func (r busTrolleyRoute) Attr() fuse.Attr {
	return fuse.Attr{
		Inode: r.inode,
		Mode:  os.ModeDir | 0555,
	}
}

// Lookup returns a node for the given bus/trolley route.
func (r busTrolleyRoute) Lookup(name string, intr fs.Intr) (n fs.Node,
	err fuse.Error) {

	log.Printf("requesting lookup for %s", name)

	switch name {
	case "locations":
		n = r.locationNode
	case "alerts":
		n = r.alertsNode
	default:
		err = fuse.ENOENT
	}

	return
}

// ReadDir returns directory entries for every SEPTA transit route.
func (busTrolleyRoute) ReadDir(intr fs.Intr) (dirs []fuse.Dirent,
	err fuse.Error) {

	locations := fuse.Dirent{Name: "locations", Type: fuse.DT_File}
	alerts := fuse.Dirent{Name: "alerts", Type: fuse.DT_File}

	dirs = append(dirs, locations, alerts)

	return
}

// busTrolleyLocation represents locations for buses and trolleys on a route.
type busTrolleyLocation struct {
	route string
	inode uint64
}

// Open sets direct IO on and returns the current busTrolleyLocation.
func (v busTrolleyLocation) Open(req *fuse.OpenRequest,
	resp *fuse.OpenResponse, intr fs.Intr) (h fs.Handle, err fuse.Error) {

	resp.Flags = resp.Flags | fuse.OpenDirectIO

	h = v

	return
}

// Attr returns attributes corresponding to the bus/trolley route.
func (v busTrolleyLocation) Attr() fuse.Attr {
	log.Printf("getting attributes for locations on %s (%d)", v.route, v.inode)
	return fuse.Attr{
		Inode: v.inode,
		Mode:  0444,
	}
}

// ReadAll connects to the SEPTA busTrolleyRoute API and returns the status of
// all vehicles on the current route.
func (v busTrolleyLocation) ReadAll(intr fs.Intr) (b []byte, err fuse.Error) {
	log.Printf("reading all for route %s", v.route)

	var ret []BusTrolley
	if ret, err = client.TransitView(v.route); err != nil {
		return
	}

	for _, bt := range ret {
		btBytes := []byte(bt.String())
		btBytes = append(btBytes, '\n')
		b = append(b, btBytes...)
	}

	return
}

// routeAlerts represents alerts for a route on any mode.
type routeAlerts struct {
	route string
	inode uint64
}

// Open sets direct IO on and returns the current routeAlerts.
func (r routeAlerts) Open(req *fuse.OpenRequest,
	resp *fuse.OpenResponse, intr fs.Intr) (h fs.Handle, err fuse.Error) {

	resp.Flags = resp.Flags | fuse.OpenDirectIO

	h = r

	return
}

// Attr returns attributes corresponding to the route.
func (r routeAlerts) Attr() fuse.Attr {
	log.Printf("getting attributes for alerts on %s (%d)", r.route, r.inode)
	return fuse.Attr{
		Inode: r.inode,
		Mode:  0444,
	}
}

func (r routeAlerts) ReadAll(intr fs.Intr) (b []byte, err fuse.Error) {
	var rts []RouteAlert
	if rts, err = client.RouteAlerts(r.route); err != nil {
		return
	}

	currentHeader := "CURRENT ALERTS:\n\n"
	advisoryHeader := "ADVISORIES:\n\n"

	b = append(b, []byte(currentHeader)...)
	for i, alert := range rts {
		altBytes := []byte(alert.CurrentMessage)
		if i+1 < len(rts) {
			altBytes = append(altBytes, '\n')
		}
		b = append(b, altBytes...)
	}

	b = append(b, []byte(advisoryHeader)...)
	for i, alert := range rts {
		altBytes := []byte(alert.AdvisoryMessage)
		if i+1 < len(rts) {
			altBytes = append(altBytes, '\n')
		}
		b = append(b, altBytes...)
	}

	return
}
