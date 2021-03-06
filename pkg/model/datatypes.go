package model

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type PairPoint2f struct {
	First  Point2f `json:"fst"`
	Second Point2f `json:"snd"`
}

type Point2f struct {
	X *float64 `json:"x"`
	Y *float64 `json:"y"`
}

type Pair2f struct {
	First  Point2f `json:"first"`
	Second Point2f `json:"second"`
}

type MapNode struct {
	Id         int     `json:"id" gorm:"primaryKey"`
	MapId      int     `json:"mapId"`
	Location   Point2f `json:"location" gorm:"embedded;embeddedPrefix:location_"`
	RootNode   bool    `json:"rootNode"`
	FloorIndex *int    `json:"floorIndex"`
	BiDirectional bool `json:"bidirectional"`
}

type NodeEdge struct {
	Id      int      `json:"id" gorm:"primaryKey"`
	MapId   int      `json:"mapId"`
	Node1Id int      `json:"node1Id"`
	Node2Id int      `json:"node2Id"`
	Node1   *MapNode `json:"-" gorm:"foreignkey:Node1Id;references:Id"`
	Node2   *MapNode `json:"-" gorm:"foreignkey:Node2Id;references:Id"`
}

type Auth0User struct {
	Id     int `gorm:"primaryKey"`
	Authid string
	Maps   []Map `gorm:"many2many:user_map_jt;"`
}

type Map struct {
	Id        int         `json:"id" gorm:"primaryKey"`
	Password  string      `json:"password"`
	Name      string      `json:"name"`
	Buildings []Building  `json:"buildings" gorm:"references:Id;OnDelete:CASCADE"`
	Users     []Auth0User `json:"-" gorm:"many2many:user_map_jt;"`
	Nodes     []MapNode   `json:"nodes" gorm:"references:Id;OnDelete:CASCADE"`
	Edges     []NodeEdge  `json:"edges" gorm:"references:Id;OnDelete:CASCADE"`
	NorthAngle float64 `json:"northAngle"`
	Deleted gorm.DeletedAt
}

type Building struct {
	Id           int     `json:"id" gorm:"primaryKey"`
	MapId        int     `json:"mapId"`
	BuildingName string  `json:"buildingName"`
	Location     Point2f `json:"location" gorm:"embedded;embeddedPrefix:location_"`
	Floors       []Floor `json:"floors" gorm:"references:Id;OnDelete:CASCADE"`
}

type Floor struct {
	Id          int      `json:"id" gorm:"primaryKey"`
	BuildingId  int      `json:"buildingId"`
	FloorNumber *int     `json:"floorNumber"`
	FloorName   string   `json:"floorName"`
	Location    Point2f  `json:"location" gorm:"embedded;embeddedPrefix:location_"`
	Rooms       []Room   `json:"rooms" gorm:"references:Id;OnDelete:CASCADE"`
	Sensors     []Sensor `json:"sensors"gorm:"OnDelete:CASCADE"`
}

type Sensor struct {
	Id int `json:"id" gorm:"primaryKey"`
	SensorId   string     `json:"sensorId"`
	FloorId    int       `json:"floorId"`
	Location   Point2f   `json:"location" gorm:"embedded;embeddedPrefix:location_"`
}

type Indent struct {
	Id         int     `json:"id" gorm:"primaryKey"`
	RoomId     int     `json:"roomId"`
	WallKeyA   *string  `json:"wallKeyA" gorm:"column:wallKeyA"`
	WallKeyB   *string  `json:"wallKeyB" gorm:"column:wallKeyB"`
	Location   float64 `json:"location"`
	Dimensions Point2f `json:"dimensions" gorm:"embedded;embeddedPrefix:size_"`
}

type Entrance struct {
	Id         int     `json:"id" gorm:"primaryKey"`
	Start Point2f `json:"start" gorm:"embedded;embeddedPrefix:start_"`
	End Point2f `json:"end" gorm:"embedded;embeddedPrefix:end_"`
}

type Room struct {
	Id         int      `json:"id" gorm:"primaryKey"`
	FloorId int      `json:"floorId"`
	Rotation   *float64 `json:"rotation"`
	Name       string   `json:"name"`
	Location   Point2f  `json:"location" gorm:"embedded;embeddedPrefix:location_"`
	Dimensions Point2f  `json:"dimensions" gorm:"embedded;embeddedPrefix:size_"`
	Indents    []Indent `json:"indents" gorm:"references:Id;OnDelete:CASCADE"`
	Polygon    []Point2f `json:"polygon" gorm:"-"`
	Walls      []*PairPoint2f `json:"walls" gorm:"-"`
	Entrances  []Entrance `json:"entrances" gorm:"references:Id;many2many:room_entrance_jt;OnDelete:CASCADE"`
}

func (r *Room) AfterFind(tx *gorm.DB) (err error) {
	r.Polygon = CalculatePolygonPoints(*r)
	r.Walls = CalculatePolygonEdgePairs(*r, false)
	json.NewEncoder(os.Stdout).Encode(r.Walls)
	json.NewEncoder(os.Stdout).Encode(r.Polygon)
	return nil
}

func PointsEqual(p1, p2 Point2f) bool {
	return *p1.X == *p2.X && *p1.Y == *p2.Y
}

func ToTsx() {
	structs := []interface{}{
		&Point2f{},
		&Map{},
		&Building{},
		&Floor{},
		&Room{},
		&Indent{},
		&Sensor{},
		&MapNode{},
		&NodeEdge{},
	}

	for _, value := range structs {
		values, name := extractStruct(value)
		val2 := generateTsxType(name, values)
		fmt.Printf("%v\n\n", val2)
	}

}

func generateTsxType(structname string, fields []string) string {
	value := "export type " + structname + " = {\n"

	for _, field := range fields {
		values := strings.Split(field, ",")
		t := convertGoTypeToTsxType(values[1])
		value += fmt.Sprintf("\t%v: %v\n", values[0], t)
	}
	value += "}"
	return value
}

func extractStruct(class interface{}) ([]string, string) {
	e := reflect.ValueOf(class).Elem()

	var values []string = []string{}

	//extract each field, and append csv's of the name & type,
	//Note: if the json tag exists on the field, the name for that will be used instead.
	for i := 0; i < e.NumField(); i++ {
		varName := e.Type().Field(i).Name
		varType := e.Type().Field(i).Type
		jsonTag := e.Type().Field(i).Tag.Get("json")

		if jsonTag == "-" {
			continue
		} else if jsonTag != "" {
			varName = jsonTag
		}
		values = append(values, fmt.Sprintf("%v,%v", varName, varType))
	}
	return values, e.Type().Name()
}

func convertGoTypeToTsxType(t string) string {
	if strings.Contains(t, ".") {
		var re = regexp.MustCompile(`[a-z]*\.`)
		if strings.Contains(t, "[]") {
			t = strings.ReplaceAll(t, "[]", "")
			t += "[]"
		}
		return re.ReplaceAllString(t, "")
	}
	t = strings.ReplaceAll(t, "*", "")

	switch t {
	case "int", "float64":
		return "number"
	case "string":
		return t
	case "bool":
		return "boolean"
	}
	panic("No Translation for type: " + t)
}

func NewPoint2f(x, y float64) Point2f {
	return Point2f{
		X: &x,
		Y: &y,
	}
}
