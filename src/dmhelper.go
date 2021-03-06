package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"bytes"
	"github.com/jamesunger/rdoclient"
	"time"
	"net"
)

var (
	addr          = flag.String("addr", ":8080", "http service address")
	assets        = flag.String("assets", defaultAssetPath(), "path to assets")
	homeTempl     *template.Template
	editPlayerTempl     *template.Template
	playerIdTempl     *template.Template
	world WorldState
)

type WorldState struct {
	Places        []Place
	Chars         []Char
	Npcs          []Char
	Place         string
	NoText        bool
	ShowParty     bool
	ShowNpcs      bool
	ShowMugs      bool
	Initiativetxt string
	Curhps        map[int]int
	Lastoutput    string
	Lastbattlemsg string
	Battlelog     string
	Outputar      []string
	Currentturn   int
	Objects       []Object
	Players	      []string
	Loggedin      []string
	Charlist      string
	Abilitymods   map[int]int
	Exptable      map[int]int
	Challtable    map[int]int
	Loggedexp     int
	Music	      string
}

type MainData struct {
	Host    string
	Content string
}

type Command struct {
	Name    string
	Args    []string
	RawArgs string
}

type Place struct {
	Name     string
	Image    string
	Desc     string
	Key      string
	Autodrop []string
	Music	 string
}

type Object struct {
	Key  string
	Name string
	Image string
	Desc string
	Contains []string
	Weight int
}

type Char struct {
	// Required fields for char or world.Npcs
	Name       string
	Class      string
	Race       string
	Abilities  Abilities
	Level      int
	InParty    bool
	Image      string
	Initiative int
	AC         int
	Alignment  string
	HP         int
	CurHP         int
	Desc       string
	Key        string
	Attacks    []Attack
	Inventory  []string


	// extra fields for player character
	Inspiration int
	Speed int
	Skills string
	ProfBonus int
	PassPerception int
	MiscProfLanguages string
	HitDice string
	PersonalityTraits string
	Background string
	Ideals string
	Bonds string
	Flaws string
	FeaturesTraits string
	Treasure string
	SpellL0 string
	SpellL1 string
	SpellL2 string
	SpellL3 string
	SpellL4 string
	SpellL5 string
	SpellL6 string
	SpellL7 string
	SpellL8 string
	SpellL9 string
	AttacksJson string
	AbilitiesJson string
	InventoryJson string
	Playername string
}

type Attack struct {
	Name       string
	Range      string
	Dtype      string
	Verb       string
	Hitbonus   int
	Damageroll string
}

type Abilities struct {
	Str int
	Dex int
	Con int
	Int int
	Wis int
	Cha int
}

func (char *Char) JsonIfy() {
	jsonatt,_ := json.Marshal(char.Attacks)
	jsoninv,_ := json.Marshal(char.Inventory)
	jsonabi,_ := json.Marshal(char.Abilities)
	char.AttacksJson = template.HTMLEscapeString(string(jsonatt))
	char.InventoryJson = template.HTMLEscapeString(string(jsoninv))
	char.AbilitiesJson = template.HTMLEscapeString(string(jsonabi))

	//fmt.Println(char.AttacksJson)
}

func (att *Attack) Display() string {
	output := ""

	output = fmt.Sprintf("<td>%s</td> <td>%d</td> <td>%s</td> <td>%s</td><td>%s</td>", att.Name, att.Hitbonus, att.Damageroll, att.Range, att.Dtype)

	return output
}

func getPlace(name string) Place {
	for i := range world.Places {
		if world.Places[i].Name == name || world.Places[i].Key == name {
			return world.Places[i]
		}
	}

	return Place{}
}

func getObject(key string) Object {
	for i := range world.Objects {
		if world.Objects[i].Key == key {
			return world.Objects[i]
		}
	}

	return Object{}
}

func getNpcOrChar(name string) Char {
	for i := range world.Npcs {
		if world.Npcs[i].Name == name {
			return world.Npcs[i]
		}

		if world.Npcs[i].Key == name {
			return world.Npcs[i]
		}

	}

	for i := range world.Chars {
		if world.Chars[i].Name == name {
			return world.Chars[i]
		}

		if world.Chars[i].Key == name {
			return world.Chars[i]
		}

	}

	return Char{}
}


func getChar(name string) Char {
	for i := range world.Chars {
		//prefix := strings.ToLower(chars[i].Name[0:3])
		if world.Chars[i].Name == name {
			return world.Chars[i]
		}

		if world.Chars[i].Key == name {
			return world.Chars[i]
		}

	}

	return Char{}
}





func prefixExists(prefix string) bool {
	for i := range world.Chars {
		if world.Chars[i].Key == prefix {
			return true
		}
	}
	return false
}

func makeCharKey(name string) string {

	prefix := strings.ToLower(name[0:3])
	prefixOrig := strings.ToLower(name[0:3])
	prefixInc := 1

	if strings.Contains(prefix, " ") {
		prefix = strings.Replace(prefix," ","-",-1)
	}

	if strings.Contains(prefix, ".") {
		prefix = strings.Replace(prefix," ","_",-1)
	}

	for {
		if prefixExists(prefix) {
			prefix = fmt.Sprintf("%s%d", prefixOrig, prefixInc)
			prefixInc = prefixInc + 1
			continue
		}
		return prefix
	}

}

func logExp(level int) {
	sendConsole(fmt.Sprintln("Players ", len(world.Players), " exp is ", world.Challtable[level]))
	world.Loggedexp = world.Loggedexp + (world.Challtable[level] / len(world.Players))
	sendConsole(fmt.Sprintln("Exp: ", world.Loggedexp))
}

func applyDamage(char Char, damage int) {
	if charIsNpc(char.Name) {
		for i := range world.Npcs {
			if char.Key == world.Npcs[i].Key {
				world.Npcs[i].CurHP = world.Npcs[i].CurHP - damage
				if world.Npcs[i].CurHP <= 0 {
					logExp(world.Npcs[i].Level)
				}
			}
		}
	} else {
		for i := range world.Chars {
			if world.Chars[i].Name == char.Name {
				world.Curhps[i] = world.Curhps[i] - damage
			}
		}
	}
}

func ReadFileContents(file *os.File) []byte {
	reader := bufio.NewReader(file)
	rawBytes, _ := ioutil.ReadAll(reader)
	return rawBytes
}

func initPlaces() {

	file, err := os.Open("assets/places.json")

	if err != nil {
		panic("Could not open assets/places.json")
	}

	filebytes := ReadFileContents(file)

	world.Places = make([]Place, 30)
	err = json.Unmarshal(filebytes, &world.Places)
	if err != nil {
		fmt.Println("Failed to read assets/places.json: ", err)
		panic(err)
	}

	//return *places
}

func initObjects() {
	file, err := os.Open("assets/objects.json")

	if err != nil {
		panic("Could not open assets/objects.json")
	}

	filebytes := ReadFileContents(file)

	//if objects == nil {
	world.Objects = make([]Object, 30)
	//}

	err = json.Unmarshal(filebytes, &world.Objects)
	if err != nil {
		fmt.Println("Failed to read assets/objects.json: ", err)
		panic(err)
	}

}

func listPlaces(filter string) {

	output := ""
	for i := range world.Places {
		//output = output + fmt.Sprintln(world.Places[i].Key, ": ", world.Places[i].Name, " - ", world.Places[i].Desc)
		if filter != "" && !strings.Contains(fmt.Sprintf("%s%s",world.Places[i].Key, world.Places[i].Name),filter) {
			continue
		} else {
			output = output + fmt.Sprintf("%s : %s\n",world.Places[i].Key, world.Places[i].Name)
		}
	}

	sendConsole(output)

}

func listObjs(filter string) {

	output := ""
	for i := range world.Objects {
		if filter != "" && !strings.Contains(fmt.Sprintf("%s%s%s",world.Objects[i].Key, world.Objects[i].Name, world.Objects[i].Desc),filter) {
			continue
		} else {
			output = output + fmt.Sprintln(world.Objects[i].Key, ": ", world.Objects[i].Name, " - ", world.Objects[i].Desc)
		}
	}
	sendConsole(output)

}





func listChars(filter string) {

	output := ""
	for i := range world.Chars {
		if filter != "" && !strings.Contains(fmt.Sprintf("%s%s",world.Chars[i].Key, world.Chars[i].Name),filter) {
			continue
		} else {
			output = output + fmt.Sprintf("%s: %s\n", world.Chars[i].Key, world.Chars[i].Name)
		}
	}
	sendConsole(output)
}

func listNpcs() {

	output := ""
	for i := range world.Npcs {
		output = output + fmt.Sprintf("(%s) %s: %d/%d\n", world.Npcs[i].Key, world.Npcs[i].Name, world.Npcs[i].CurHP, world.Npcs[i].HP)
	}
	sendConsole(output)
}

func cloneChar(char Char) Char {
	nchar := Char{}

	nchar.Name = char.Name
	nchar.Class = char.Class
	nchar.Race = char.Race
	nchar.Abilities = char.Abilities
	nchar.Level = char.Level
	nchar.InParty = char.InParty
	nchar.Image = char.Image
	nchar.Initiative = char.Initiative
	nchar.AC = char.AC
	nchar.Alignment = char.Alignment
	nchar.HP = char.HP
	nchar.CurHP = char.HP
	nchar.Desc = char.Desc
	nchar.Key = makeCharKey(char.Name)
	nchar.Attacks = char.Attacks
	nchar.Inventory = char.Inventory

	return nchar
}

func initChars(wipehps bool) {

	file, err := os.Open("assets/chars.json")

	if err != nil {
		panic("Could not open assets/chars.json")
	}

	filebytes := ReadFileContents(file)
	//if chars == nil {
	world.Chars = make([]Char, 30)
	//nchars = make([]Char,30)
	//}
	err = json.Unmarshal(filebytes, &world.Chars)
	if err != nil {
		fmt.Println("Failed to read assets/chars.json: ", err)
		panic(err)
	}

	for i := range world.Players {
		char := loadPlayerChar(world.Players[i])
		char.Playername = world.Players[i]
		fmt.Println("Loaded ", char.Name)
		world.Chars = append(world.Chars,char)
	}

	//if curhps != nil {
	//	curhps = make(map[int]int)
	//}

	for i := range world.Chars {
		if wipehps {
			world.Curhps[i] = world.Chars[i].HP
		}
		world.Chars[i].Key = makeCharKey(world.Chars[i].Name)

		/*if chars[i].NpcInstances > 0 {
			for k := 0; k <= chars[i].NpcInstances; k++ {
				nchars[k
			}
		}*/
	}

}

func initNpcs() {
	world.Npcs = nil
	//world.Npcs = new([]Char)
}

func defaultAssetPath() string {
	p, err := build.Default.Import(".", "", build.FindOnly)
	if err != nil {
		return "."
	}
	return p.Dir
}

func alreadyLoggedIn(playername string) bool {
	for i := range world.Loggedin {
		if world.Loggedin[i] == playername {
			return true
		}
	}

	return false
}

func validatePlayer(player string) bool {
	if player == "ohgodmedusa" {
		return true
	}

	for i := range world.Players {
		if strings.ToLower(player) == world.Players[i] {
			//if alreadyLoggedIn(players[i]) {
			//	fmt.Println("Already logged in! ", players[i])
			//	return false
			//}
			fmt.Println("Fantastic, you are", player)
			world.Loggedin = append(world.Loggedin,world.Players[i])
			return true
		}
	}

	return false
}

func homeHandler(c http.ResponseWriter, req *http.Request) {

	req.ParseForm()
	cv,_ := req.Cookie("playername")
	//if cv != nil && cv.Value != "" && !alreadyLoggedIn(cv.Value) {
	//		http.SetCookie(c,&http.Cookie{Name: "playername", Value: "", MaxAge: -1})
	//}

	if (cv == nil || cv.Value == "") && len(req.Form["playername"]) == 0 {
		playerIdTempl.Execute(c,nil)
		return
	}


	if strings.Contains(req.URL.Path, "assets") {
		//fmt.Println("Serving assets...")
		chttp.ServeHTTP(c, req)

	} else if strings.Contains(req.URL.Path, "logout") {
		http.SetCookie(c,&http.Cookie{Name: "playername", Value: "", MaxAge: -1})
		http.Redirect(c,req,"/",302)
	} else if strings.Contains(req.URL.Path, "attack") {
		issueAttack(c, req)
	} else if strings.Contains(req.URL.Path, "char") {
		webViewChar(c, req)
	} else if strings.Contains(req.URL.Path, "playeredit") {
		fmt.Println("Player edit...")
		editPlayerChar(c, req)
	} else if strings.Contains(req.URL.Path, "playerid") {
		if len(req.Form["playername"]) == 1 && validatePlayer(req.Form["playername"][0]) {
			http.SetCookie(c,&http.Cookie{Name: "playername", Value: req.Form["playername"][0]})
			http.Redirect(c,req,"/",302)
			fmt.Println("Got cookie, redirecting.")
			//mainData := MainData{Host: req.Host, Content: lastoutput}
			//homeTempl.Execute(c, mainData)
		} else if cv != nil && cv.Value != "" {
			http.Redirect(c,req,"/",302)
			//mainData := MainData{Host: req.Host, Content: fmt.Sprintf("You are already logged in as %s", cv.Value)}
			//homeTempl.Execute(c, mainData)
		} else {
			//http.SetCookie(c,&http.Cookie{Name: "playername", Value: "", MaxAge: -1})
			//http.Redirect(c,req,"/",302)
			playerIdTempl.Execute(c,nil)
		}
	} else {
		mainData := MainData{Host: req.Host, Content: world.Lastoutput}
		homeTempl.Execute(c, mainData)
	}

}

func parseInput(origin, input string) *Command {
	input = strings.TrimRight(input, "\n")
	parts := strings.Split(input, " ")
	cmd := &Command{}
	if len(parts) < 1 {
		sendConsole(fmt.Sprintln("Invalid command."))
		return &Command{Name: "Invalid command"}
	} else if len(parts) >= 1 {
		cmd = &Command{Name: parts[0], Args: parts[1:]}
		cmd.RawArgs = strings.Join(parts[1:], " ")
	}

	//fmt.Println(cmd)
	//fmt.Println("Raw: ", cmd.RawArgs)
	//fmt.Println("Args: ", parts)
	//fmt.Println("CMD: ", cmd.Name)

	// Send all commands
	//if cmd.Name != "" {
	//	sendConsole(fmt.Sprintf("(%s) -> %s %s\n",origin, cmd.Name, cmd.RawArgs))
	//}
	return cmd
}

func getNpcTxt() string {
	output := ""

	if !world.ShowNpcs {
		return output
	}

	for i := range world.Npcs {
		if world.Npcs[i].CurHP <= 0 {
			world.Npcs[i].Image = "/assets/skull.jpg"
		}
		if world.ShowMugs {
			//output = output + fmt.Sprintf("<div class=\"npc\"><a href=\"/char?name=%s\"><img src=\"%s\" width=180/></a><br><b>%s (%s)</b><br>%s</div>  ", world.Npcs[i].Name, world.Npcs[i].Image, world.Npcs[i].Name, world.Npcs[i].Key, world.Npcs[i].Race)
			output = output + renderChar(world.Npcs[i])
		} else {
			output = output + fmt.Sprintf("<div class=\"npcnoimg\"><b>%s</b><br>%s   </div>", world.Npcs[i].Name, world.Npcs[i].Race)
		}
	}

	//fmt.Println(output)
	return output

}


func renderChar(char Char) string {
	output := ""
	if charIsNpc(char.Name) {
		wounded := ""
		perc := float32(char.CurHP)/float32(char.HP)
		//fmt.Println("Perc: ", perc)
		if perc == 1.0 {
			wounded = ""
		} else if perc > 0.80 {
			wounded = "green"
		} else if perc > 0.30 {
			wounded = "yellow"
		} else if perc < 0.30 {
			wounded = "red"
		}


		if char.CurHP <= 0 {
			char.Image = "/assets/skull.jpg"
		}
		output = fmt.Sprintf("<div class=\"npc\"><a href=\"/char?name=%s\"><img src=\"%s\" width=180/></a><br><b><span style=\"color: %s\">%s (%s)</span></b><br>%s</div>  ", char.Name, char.Image, wounded, char.Name, char.Key, char.Race)
	} else {
		curhp := getHP(char.Key)
		output = fmt.Sprintf("<div id=\"%s\" class=\"partymember\"><div><a href=\"/char?name=%s\"><img src=\"%s\" width=180/></a></div><b>%s</b><br>%s/%s/%d<br>%d/%d   </div>", char.Name, char.Name, char.Image, char.Name, char.Race, char.Class, char.Level, curhp, char.HP)
	}
	return output
}

func renderNpcChar(char Char) string {
	output := ""
	output = fmt.Sprintf("<div class=\"npc\"><a href=\"/char?name=%s\"><img src=\"%s\" width=180/></a><br><b>%s (%s)</b><br>%s</div>  ", char.Name, char.Image, char.Name, char.Key, char.Race)

	return output
}


func renderPChar(char Char, curhp int) string {
	output := ""
	output = fmt.Sprintf("<div id=\"%s\" class=\"partymember\"><div><a href=\"/char?name=%s\"><img src=\"%s\" width=180/></a></div><b>%s</b><br>%s/%s/%d<br>%d/%d   </div>", char.Name, char.Name, char.Image, char.Name, char.Race, char.Class, char.Level, curhp, char.HP)

	return output
}


func renderParty() string {

	output := ""

	if !world.ShowParty {
		return output
	}

	for i := range world.Chars {
		curhp := world.Curhps[i]
		if world.Chars[i].InParty {

			if curhp < 0 {
				world.Chars[i].Image = "/assets/skull.jpg"
			}

			if world.ShowMugs {
				//output = output + fmt.Sprintf("<div id=\"%s\" class=\"partymember\"><div><a href=\"/char?name=%s\"><img src=\"%s\" width=180/></a></div><b>%s</b><br>%s/%s/%d<br>%d/%d   </div>", chars[i].Name, chars[i].Name, chars[i].Image, chars[i].Name, chars[i].Race, chars[i].Class, chars[i].Level, curhp, chars[i].HP)
				output = output + renderChar(world.Chars[i])
			} else {
				output = output + fmt.Sprintf("<div id=\"%s\" class=\"partymembernoimg\"><b>%s</b><br>%s/%s/%d<br>%d/%d   </div>", world.Chars[i].Name, world.Chars[i].Name, world.Chars[i].Race, world.Chars[i].Class, world.Chars[i].Level, curhp, world.Chars[i].HP)
			}
		}
	}

	return output
}

func renderContent(msg string, cmd *Command) string {
	cplace := getPlace(world.Place)
	imagetxt := "<script type=\"text/javascript\">$(\"#picture\").text(\"\");"
	if cplace.Image != "" {
		//imagetxt = fmt.Sprintf("<script type=\"text/javascript\">$(\"#picture\").text(\"\");$(\"#picture\").append(\"<img height=768 src='%s'/>\");", cplace.Image)
		imagetxt = fmt.Sprintf("<script type=\"text/javascript\">$(\"#picture\").text(\"\");$(\"#picture\").append(\"<img height=1000 src='%s'/>\");", cplace.Image)
	}

	if world.Initiativetxt != "" && cmd.Name != "v" && cmd.Name != "vo" && cmd.Name != "blog" && cmd.Name != "att" && cmd.Name != "ant" && cmd.Name != "msg" {
	//if world.Initiativetxt != "" && cmd.Name != "v" && cmd.Name != "vo" && cmd.Name != "blog" {
		world.Initiativetxt = renderInitiativeTxt(world.Outputar)
		//msg = fmt.Sprintf("<div id=\"initiative\"><span id=\"initiativetxt\">%s   <audio autoplay loop><source src=\"/assets/fight_real.ogg\" type=\"audio/ogg\">Your browser does not support the audio element.</audio> </span></div><div id=\"msgtxt\">%s</div>", world.Initiativetxt, msg)
		msg = fmt.Sprintf("<div id=\"initiative\"><span id=\"initiativetxt\">%s  </span></div><div id=\"msgtxt\">%s</div>", world.Initiativetxt, msg)
	}

	placedesc := cplace.Desc
	npctxt := getNpcTxt()
	content := ""
	if cmd.Name == "att" || cmd.Name == "ant" {
		cchar := getCharAttacker(msg)
		//cchar := getCharTurn(currentturn-1)
		//tchar := getCharTarget(msg)
		tchar := Char{}
		if len(cmd.Args) == 0 {
			tchar = getCharTarget(msg)
		} else {
			tchar = getNpcOrChar(cmd.Args[1])
		}


		if tchar.Name == "" || cchar.Name == "" {
			content = fmt.Sprintf("<div id=\"mainarea\"><div id=\"title\">%s</div><div id=\"msg\">%s</div>  </div>    <div class=\"flex-container\" id=\"party\">%s</div>  %s$(\"#picture\").css(\"opacity\", \".47\");</script>", cplace.Name, msg, renderParty(), imagetxt)
		} else {


			world.Initiativetxt = renderInitiativeTxt(world.Outputar)
			//content = fmt.Sprintf("<div id=\"mainarea\"><div id=\"title\">%s</div>%s<div id=\"msg\">%s</div>%s  </div>    <div class=\"flex-container\" id=\"party\">%s</div>  %s$(\"#picture\").css(\"opacity\", \".47\");</script>", cplace.Name, renderChar(cchar), msg, renderChar(tchar), renderParty(), imagetxt)
			//content = fmt.Sprintf("<div id=\"mainarea\"><a class=\"js-open-modal\" id=\"foobar\" href=\"#\" data-modal-id=\"popup\"></a><div id=\"title\">%s</div><div id=\"desc\">%s</div><div id=\"initiative\"><span id=\"initiativetxt\">%s</span></div> <div class=\"flex-container\" id=\"npcs\">%s</div><div id=\"popup\" class=\"fight-box\"> <header>%s vs %s</header> <div class=\"modal-body\">     %s<div id=\"msg\">%s</div>%s  </div> <footer/> </div> </div>    <div class=\"flex-container\" id=\"party\">%s</div>  %s$(\"#picture\").css(\"opacity\", \".47\");  var appendthis =  (\"<div class='modal-overlay js-modal-close'></div>\"); $(\"body\").append(appendthis); $(\".modal-overlay\").fadeTo(800, 0.7); var modalBox = $('#foobar').attr('data-modal-id'); $('#'+modalBox).fadeIn($('#foobar').data()); </script>", cplace.Name, placedesc, world.Initiativetxt, npctxt, cchar.Name, tchar.Name, renderChar(cchar), msg,renderChar(tchar), renderParty(), imagetxt)
			content = fmt.Sprintf("<a class=\"js-open-modal\" id=\"foobar\" href=\"#\" data-modal-id=\"popup\"></a><div id=\"mainarea\"><div id=\"title\">%s</div><div id=\"desc\">%s</div> <div id=\"msg\"><div id=\"initiative\"><span id=\"initiativetxt\">%s</span></div>  </div> <div class=\"flex-container\" id=\"npcs\">%s</div>     <div id=\"popup\" class=\"fight-box\"> <header>%s vs %s</header> <div class=\"modal-body\">%s <div id=\"battlemsg\">%s</div>    %s  </div> <footer/> </div> </div>    <div class=\"flex-container\" id=\"party\">%s</div>  %s$(\"#picture\").css(\"opacity\", \".47\");  var appendthis =  (\"<div class='modal-overlay js-modal-close'></div>\"); $(\"#mainarea\").append(appendthis); /*$(\".modal-overlay\").fadeTo(500, 1);*/ $(\".modal-overlay\").css(\"opacity\",\".90\"); var modalBox = $('#foobar').attr('data-modal-id'); $('#'+modalBox).fadeIn(\"slow\",\"swing\",$('#foobar').data()); </script>", cplace.Name, placedesc,  world.Initiativetxt,  npctxt, cchar.Name, tchar.Name, renderChar(cchar), msg, renderChar(tchar), renderParty(), imagetxt)
			//content = fmt.Sprintf("<a class=\"js-open-modal\" id=\"foobar\" href=\"#\" data-modal-id=\"popup\"></a><div id=\"mainarea\"><div id=\"title\">%s</div><div id=\"desc\">%s</div> <div id=\"msg\"><div id=\"initiative\"><span id=\"initiativetxt\">%s</span></div></div><div id=\"popup\" class=\"fight-box\"> <div class=\"modal-body\">%s v %s&nbsp;%s <div id=\"battlmsg\">%s</div>    %s  </div>  </div> </div> <div class=\"flex-container\" id=\"npcs\">%s</div>   <div class=\"flex-container\" id=\"party\">%s</div>  %s$(\"#picture\").css(\"opacity\", \".47\");  var appendthis =  (\"<div class='modal-overlay js-modal-close'></div>\"); $(\"body\").append(appendthis); $(\".modal-overlay\").fadeTo(800, 0.7); var modalBox = $('#foobar').attr('data-modal-id'); $('#'+modalBox).fadeIn($('#foobar').data()); </script>", cplace.Name, placedesc,  world.Initiativetxt,  cchar.Name, tchar.Name, renderChar(cchar), msg,  renderChar(tchar), npctxt,renderParty(), imagetxt)
		}
	} else if cmd.Name == "msg" {
			content = fmt.Sprintf("<a class=\"js-open-modal\" id=\"foobar\" href=\"#\" data-modal-id=\"popup\"></a><div id=\"mainarea\"><div id=\"title\">%s</div><div id=\"desc\">%s</div> <div id=\"msg\">  </div> <div class=\"flex-container\" id=\"npcs\">%s</div>     <div id=\"popup\" class=\"fight-box\"> <div class=\"modal-body\"><span id=\"modalmsg\">%s</span>    </div> </div> </div>    <div class=\"flex-container\" id=\"party\">%s</div>  %s$(\"#picture\").css(\"opacity\", \".47\");  var appendthis =  (\"<div class='modal-overlay js-modal-close'></div>\"); $(\"#mainarea\").append(appendthis); /*$(\".modal-overlay\").fadeTo(500, 1);*/ $(\".modal-overlay\").css(\"opacity\",\".90\"); var modalBox = $('#foobar').attr('data-modal-id'); $('#'+modalBox).fadeIn(\"slow\",\"swing\",$('#foobar').data()); </script>", cplace.Name, placedesc,  npctxt, msg, renderParty(), imagetxt)
	} else {
		content = fmt.Sprintf("<div id=\"mainarea\"><div id=\"title\">%s</div><div id=\"desc\">%s</div><div id=\"msg\">%s</div><div class=\"flex-container\" id=\"npcs\">%s</div>  </div>    <div class=\"flex-container\" id=\"party\">%s</div>  %s$(\"#picture\").css(\"opacity\", \".47\");$(\".fight-box, .modal-overlay\").fadeOut(500, function() { $(\".modal-overlay\").remove();});</script>", cplace.Name, placedesc, msg, npctxt, renderParty(), imagetxt)
	}


	if world.NoText {
		content = fmt.Sprintf("%s$(\"#picture\").css(\"opacity\", \"1\");</script>", imagetxt)
	}

	if world.Music != "" {
		content = content + fmt.Sprintf("<!-- MUSIC: '%s' -->", world.Music)
	}

	if cmd.Name == "v" || cmd.Name == "vo" || cmd.Name == "blog" {
		content = fmt.Sprintf("<div id=\"msg\">%s</msg>", msg)
	}

	// DEBUG
	//fmt.Println(content)
	return content
}


func parseDiceString(dicestring string) (string,string,string) {
	numdies := dicestring[0]
	dicesize := ""
	bonus := ""
	//fmt.Println("num dies", fmt.Sprintf("%c",numdies))
	indexpl := strings.Index(dicestring,"+")
	if indexpl == -1 {
		dicesize = dicestring[2:]
	} else {
		dicesize = dicestring[2:indexpl]
		bonus = dicestring[indexpl+1:]
	}

	return fmt.Sprintf("%c",numdies), dicesize, bonus
}

func getROCDiceResults(dicestring string) string {
	roll := 0
	numdies,dicesize,bonus := parseDiceString(dicestring)


	numdiesi,_ := strconv.Atoi(numdies)
	dicesizei,_ := strconv.Atoi(dicesize)
	myints,err := rdoclient.GenerateIntegers("26a65c82-7091-45f7-af12-414589392fb0",numdiesi,1,dicesizei,true)

	if err != nil {
		fmt.Println("WTF, failed to get random number:",err)
		return "0"
	}

	for i := range myints {
		roll = roll + myints[i]
	}

	if bonus != "" {
		intgr,_ := strconv.Atoi(bonus)
		roll = roll + intgr
	}

	return fmt.Sprintf("%d",roll)

}

func getRODiceResults(dicestring string) string {
	roll := 0
	numdies,dicesize,bonus := parseDiceString(dicestring)
	//cmd := exec.Command("curl", fmt.Sprintf("https://www.random.org/integers/?num=%s&min=1&max=%s&col=4&base=10&format=plain&md=new",numdies,dicesize))
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("https://www.random.org/integers/?num=%s&min=1&max=%s&col=4&base=10&format=plain&md=new",numdies,dicesize))
	body, _ := ioutil.ReadAll(resp.Body)
	

	//fmt.Println("curl", fmt.Sprintf("https://www.random.org/integers/?num=%s&min=1&max=%s&col=4&base=10&format=plain&md=new",numdies,dicesize))
	//results, err := cmd.Output()
	//if err != nil {
	//	fmt.Println("Failed to fetch random nums from random.org", err)
	//	return ""
	//}

	results := bytes.NewBuffer(body).String()

	resultss := strings.TrimRight(string(results), "\t")
	resultss = strings.TrimRight(resultss, "\n")
	fmt.Println("Random org output: ", resultss)

	if numdies == "1" {
		roll,err = strconv.Atoi(resultss)
		if err != nil {
			fmt.Println("Failed to parse", resultss, " got ", err)
		}
		//fmt.Println("Got roll", roll)
	} else {
		parts := strings.Split(resultss,"\t")
		for p := range parts {
			intgr,err := strconv.Atoi(parts[p])
			if err != nil {
				fmt.Println("Failed to parse", resultss, " got ", err)
			}

			roll = roll + intgr
			//fmt.Println("Got part roll:", roll)
		}
		//fmt.Println("Got final roll", roll)
	}

	if bonus != "" {
		//fmt.Println("Yep a bonus")
		intgr,_ := strconv.Atoi(bonus)
		roll = roll + intgr
	}



	return fmt.Sprintf("%d",roll)
	//return getLocalDiceResults(dicestring)
}


func getLocalDiceResults(dicestring string) string {
	numdies,dicesize,bonus := parseDiceString(dicestring)
	fmt.Println(numdies,dicesize,bonus)
	//cmd := exec.Command("rolldice", "-r", dicestring)
	cmd := exec.Command("rolldice", dicestring)
	results, err := cmd.Output()
	if err != nil {
		fmt.Println(fmt.Sprintf("rolldice -r %s", dicestring), results, " with err ", err)
	}
	//fmt.Println(dicestring,"...",string(results))
	return string(results)
}

func getDiceResults(dicestring string) string {
	// local system spawn with dicer roller
	return getLocalDiceResults(dicestring)

	// random org client
	//return getROCDiceResults(dicestring)

	// random org wget
	//return getRODiceResults(dicestring)
}

func charIsNpc(name string) bool {
	for i := range world.Npcs {
		if world.Npcs[i].Name == name || world.Npcs[i].Key == name {
			return true
		}
	}

	return false
}

func renderInitiativeTxt(ar []string) string {
	output := ""
	for i := range world.Outputar {
		if ar[i] == "" {
			continue;
		}
		if i == world.Currentturn {
			output = fmt.Sprintf("%s<span id=\"currentturn\">%s</span><br>", output, ar[i])
		} else {
			output = fmt.Sprintf("%s%s<br>", output, ar[i])
		}
	}


	return output
}

func getCharTurn(turn int) Char {
	nchar := Char{}
	for k := range world.Chars {
		if (world.Chars[k].InParty || charIsNpc(world.Chars[k].Name)) {
			if strings.Contains(world.Outputar[turn], world.Chars[k].Name) {
				nchar = world.Chars[k]
				//return chars[k]
			}
		}
	}

	return nchar
}

func getCharTarget(bmsg string) Char {
	nchar := Char{}
	for k := range world.Chars {
		if (world.Chars[k].InParty || charIsNpc(world.Chars[k].Name)) {
			if strings.Contains(bmsg, fmt.Sprintf("attacks %s",world.Chars[k].Name)) {
				nchar = world.Chars[k]
				//return chars[k]
			}
		}
	}
	return nchar
}

func getCharAttacker(bmsg string) Char {
	nchar := Char{}
	for k := range world.Chars {
		if (world.Chars[k].InParty || charIsNpc(world.Chars[k].Name)) {
			if strings.Contains(bmsg, fmt.Sprintf("%s attacks",world.Chars[k].Name)) {
				nchar = world.Chars[k]
				//return chars[k]
			}
		}
	}
	return nchar
}






func rollInitiatives(advantages string) string {
	//output := "<b>Initiative</b>"
	output := ""

	advs := strings.Split(advantages, " ")

	rolls := make(map[int]int)
	var values []int
	//var outputar []string
	csize := 0

	alreadyrolled := make(map[string]bool)

	for i := range world.Chars {
		//fmt.Println(i, " for ", chars[i].Name)
		if world.Chars[i].InParty || charIsNpc(world.Chars[i].Name) {
			_,e := alreadyrolled[world.Chars[i].Name]
			if e == true {
				fmt.Println("Already rolled for", world.Chars[i].Name)
				continue
			}
			sendConsole(fmt.Sprintln("Rolling init for...", world.Chars[i].Name))
			alreadyrolled[world.Chars[i].Name] = true
			csize++
			//dres := getDiceResults(fmt.Sprintf("1d20+%d",chars[i].Initiative))
			dres := getDiceResults("1d20")
			dres = strings.TrimRight(dres, " \n")
			init, err := strconv.Atoi(dres)
			init = init + world.Chars[i].Initiative

			for k := range advs {
				ap := strings.Split(advs[k], "=")
				if len(ap) < 2 {
					break
				}

				if ap[0] == world.Chars[i].Key {
					dres2 := getDiceResults("1d20")
					dres2 = strings.TrimRight(dres2, " \n")
					init2, _ := strconv.Atoi(dres2)
					init2 = init2 + world.Chars[i].Initiative
					if ap[1] == "adv" {
						sendConsole(fmt.Sprintf("%s advantage %d %d\n", world.Chars[i].Name, init, init2))
						if init2 >= init {
							init = init2
						}
					} else if ap[1] == "dis" {
						sendConsole(fmt.Sprintf("%s disadvantage %d %d\n", world.Chars[i].Name, init, init2))
						if init2 <= init {
							init = init2
						}
					}
					break
				}
			}

			if err != nil {
				sendConsole(fmt.Sprintln("Could not convert ", dres, " to int: ", err))
			}
			rolls[i] = init
			values = append(values, init)
		}
	}

	world.Outputar = make([]string, 1)
	sort.Sort(sort.Reverse(sort.IntSlice(values)))
	for i := range values {
	CHAR:
		for k := range world.Chars {
			if (world.Chars[k].InParty || charIsNpc(world.Chars[k].Name)) && rolls[k] == values[i] {
				for j := range world.Outputar {
					if strings.Contains(world.Outputar[j], world.Chars[k].Name) {
						continue CHAR
					}
				}
				world.Outputar = append(world.Outputar, fmt.Sprintf("%s (%d)", world.Chars[k].Name, values[i]))
				sendConsole(fmt.Sprintf("%s - %s (%d)\n", world.Chars[k].Key, world.Chars[k].Name, values[i]))
				//output = fmt.Sprintf("%s<br>%s (%d)",output,chars[k].Name, values[i])
			}
		}
	}

	world.Currentturn = 0
	nextTurn()
	output = renderInitiativeTxt(world.Outputar)

	return output
}

func attack(char1 Char, atti int, char2 Char, adv string) string {
	if atti >= len(char1.Attacks) {
		sendConsole(fmt.Sprintln("Invalid attack index ", atti))
		return "";
	}
	genericMsg := fmt.Sprintf("%s attacks %s with %s<br>\n", char1.Name, char2.Name, char1.Attacks[atti].Name)
	//hitroll := getDiceResults(fmt.Sprintf("1d20+%d",char1.Attacks[atti].Hitbonus))
	attackstring := ""

	hitroll := getDiceResults("1d20")
	hitroll = strings.TrimRight(hitroll, " \n")
	hr, _ := strconv.Atoi(hitroll)
	hrb := hr + char1.Attacks[atti].Hitbonus
	hitroll2 := getDiceResults("1d20")
	hitroll2 = strings.TrimRight(hitroll2, " \n")
	hr2, _ := strconv.Atoi(hitroll2)
	hrb2 := hr2 + char1.Attacks[atti].Hitbonus
	if adv == "adv" {
		attackstring = fmt.Sprintf("Advantage attack roll: 1d20+%d (%d %d) vs AC %d: ", char1.Attacks[atti].Hitbonus, hrb, hrb2, char2.AC)
		if hr2 >= hr {
			hr = hr2
			hrb = hrb2
		}
	} else if adv == "dis" {
		attackstring = fmt.Sprintf("Disadvantage attack roll: 1d20+%d (%d %d) vs AC %d: ", char1.Attacks[atti].Hitbonus, hrb, hrb2, char2.AC)
		if hr2 <= hr {
			hr = hr2
			hrb = hrb2
		}
	} else {
		attackstring = fmt.Sprintf("Attack roll: 1d20+%d (%d) vs AC %d: ", char1.Attacks[atti].Hitbonus, hrb, char2.AC)
	}

	damageroll := getDiceResults(char1.Attacks[atti].Damageroll)
	damageroll = strings.TrimRight(damageroll, " \n")
	dr, _ := strconv.Atoi(damageroll)
	battlemsg := ""
	if hr == 20 {
		applyDamage(char2, dr)
		attackstring = attackstring + " <b>CRITICAL HIT!</b><br>\n"
		damagestring := fmt.Sprintf("%s savagely %s %s for %d damage.<br>\nDamage Roll: %s (%d)", char1.Name, char1.Attacks[atti].Verb, char2.Name, dr, char1.Attacks[atti].Damageroll, dr)
		battlemsg = genericMsg + attackstring + damagestring
	} else if hr == 1 {
		battlemsg = genericMsg + attackstring + " <b> CRITICAL MISS!</b>\n"
	} else if hrb >= char2.AC {
		applyDamage(char2, dr)
		attackstring = attackstring + " <b>HIT!</b><br>\n"
		damagestring := fmt.Sprintf("%s %s %s for %d damage.<br>\nDamage Roll: %s (%d)", char1.Name, char1.Attacks[atti].Verb, char2.Name, dr, char1.Attacks[atti].Damageroll, dr)
		battlemsg = genericMsg + attackstring + damagestring
	} else {
		battlemsg = genericMsg + attackstring + " <b>MISS!</b>"
	}

	sendConsole(fmt.Sprintln(battlemsg))
	world.Lastbattlemsg = ""
	fbattlemsg := fmt.Sprintf("<span class=\"blog\">%s</span><br>", world.Lastbattlemsg) + battlemsg
	world.Lastbattlemsg = battlemsg
	world.Battlelog = world.Battlelog + "<br>" + battlemsg
	return fbattlemsg
}

func viewObjConsole(key string) string {
        obj := getObject(key)                                                           
        output,err := json.MarshalIndent(obj,""," ")                                                 
        if err != nil {                                                                 
                fmt.Println("WTF error:",err)                                           
                return ""                                                               
	}
	return string(output)
}

func viewCharConsole(name string) string {
	char := getChar(name)
        output,err := json.MarshalIndent(char,""," ")
        if err != nil {
                fmt.Println("WTF error:",err)
                return ""
	}
	return string(output)
}

func viewObj(key string) string {
	output := ""
	obj := getObject(key)
	output = fmt.Sprintf("<div id=\"viewobj\"><img id=\"objimg\" src=\"%s\"/></div><div id=\"objinfo\"><p>%s</p><p>%s</p>",  obj.Image, obj.Name, obj.Desc)
	return output
}

func viewChar(name string, showstats bool) string {
	output := ""
	char := getChar(name)

	attacks := "<table><tr><td>name</td>  <td>to hit</td>  <td>dmg</td>  <td>range</td><td>type</td></tr>"
	for i := range char.Attacks {
		attacks = attacks + "<tr>\n" + char.Attacks[i].Display() + "</tr>"
	}
	attacks = attacks + "</table>"

	inventory := "<br>Inventory<br>"
	for i := range char.Inventory {
		obj := getObject(char.Inventory[i])
		inventory = inventory + obj.Name + " (" + obj.Key + ")<br>"
	}

	if showstats {

		output = fmt.Sprintf("<div id=\"viewchar\"><img id=\"charimg\" src=\"%s\"/></div><div id=\"charinfo\"><p>%s, Level %d %s</p><p>%s</p>Str: %d (%d) Dex: %d (%d) Con: %d (%d) Int: %d (%d) Wis: %d (%d) Cha: %d (%d) <br>Initiative: %d<br>AC: %d<br>HP: %d<br>Alignment: %s<br>Attacks:<br> %s%s", char.Image, char.Name, char.Level, char.Class, char.Desc, char.Abilities.Str, world.Abilitymods[char.Abilities.Str], char.Abilities.Dex, world.Abilitymods[char.Abilities.Dex], char.Abilities.Con, world.Abilitymods[char.Abilities.Con], char.Abilities.Int, world.Abilitymods[char.Abilities.Int], char.Abilities.Wis, world.Abilitymods[char.Abilities.Wis], char.Abilities.Cha, world.Abilitymods[char.Abilities.Cha], char.Initiative, char.AC, char.HP, char.Alignment, attacks, inventory)
		output = output + fmt.Sprintf("<hr>Inspiration: %d<br> Proficiency Bonus: %d<br> Passive Perception: %d<br> Hit Dice: %s<br> Speed: %d<br> Skills: %s<br><hr>Misc Proficiencies and Languages: %s<br> Personality Traits: %s<br> Ideals: %s<br> Bonds: %s<br> Flaws: %s<br> Features and Traits: %s<br> Treasure: %s<br> Spells<br> Level 0: %s<br> Level 1: %s<br> Level 2: %s<br> Level 3: %s<br> Level 4: %s<br> Level 5: %s<br> Level 6: %s<br> Level 7: %s<br> Level 8: %s<br> Level 9: %s<br> </div>", char.Inspiration, char.ProfBonus, char.PassPerception, char.HitDice, char.Speed, char.Skills, char.MiscProfLanguages, char.PersonalityTraits, char.Ideals, char.Bonds, char.Flaws, char.FeaturesTraits, char.Treasure, char.SpellL0, char.SpellL1, char.SpellL2, char.SpellL3, char.SpellL4, char.SpellL5, char.SpellL6, char.SpellL7, char.SpellL8, char.SpellL9)
	} else {
		output = fmt.Sprintf("<div id=\"viewchar\"><img id=\"charimg\" src=\"%s\"/></div><div id=\"charinfo\"><p>%s</p><p>%s</p>", char.Image, char.Name, char.Desc)
	}

	//fmt.Println(char.Name)
	//fmt.Println(output)
	return output
}


/*
func viewNpcChar(name string) string {
	output := ""
	char := getChar(name)

/*	attacks := "<table><tr><td>name</td>  <td>to hit</td>  <td>dmg</td>  <td>type</td></tr>"
	for i := range char.Attacks {
		attacks = attacks + "<tr>\n" + char.Attacks[i].Display() + "</tr>"
	}
	attacks = attacks + "</table>"
	*/

//	output = fmt.Sprintf("<div id=\"viewchar\"><img id=\"charimg\" src=\"%s\"/></div><div id=\"charinfo\"><p>%s</p><p>%s</div>", char.Image, char.Name, char.Desc)

	//fmt.Println(char.Name)
	//fmt.Println(output)
//	return output
//}

func dropNpc(name string) {
	char := getChar(name)
	if char.Name == "" {
		sendConsole(fmt.Sprintln("Invalid charname"))
		return
	}
	nchar := cloneChar(getChar(name))
	world.Chars = append(world.Chars, nchar)
	world.Npcs = append(world.Npcs, nchar)
	sendConsole(fmt.Sprintln("Dropped ", nchar.Key))
}

func syncNpcs() {
	for i := range world.Npcs {
		world.Chars = append(world.Chars,world.Npcs[i])
	}
}

func setHP(name string, hp int) {
	if charIsNpc(name) {
		for i := range world.Npcs {
			if name == world.Npcs[i].Key {
				world.Npcs[i].CurHP = hp
				//fmt.Println(hp)
				if hp <= 0 {
					logExp(world.Npcs[i].Level)
				}
			}
		}
	} else {
		for i := range world.Chars {
			if world.Chars[i].Key == name {
				world.Curhps[i] = hp
			}
		}
	}
}

func getHP(name string) int {
	if charIsNpc(name) {
		for i := range world.Npcs {
			if name == world.Npcs[i].Key {
				return world.Npcs[i].CurHP
			}
		}
	} else {
		for i := range world.Chars {
			if world.Chars[i].Key == name {
				return world.Curhps[i]
			}
		}
	}

	return 0
}

func nextTurn() {
	if world.Currentturn < len(world.Outputar)-1 {
		world.Currentturn++
	} else {
		world.Currentturn = 1
	}
}

func prevTurn() {
	if world.Currentturn > 1 {
		world.Currentturn--
	}
}

func printStatus() {
	sendConsole(fmt.Sprintln("Place: ", world.Place))
	sendConsole(fmt.Sprintln("Exp: ", world.Loggedexp))
	listNpcs()
	if world.Initiativetxt != "" {
		sendConsole(fmt.Sprintln(world.Initiativetxt))
	}
}

func getCharWithTurn() Char {
	//for i := range outputar {
		//sendConsole(fmt.Sprintln("Current turn: ", world.Currentturn))
		if len(world.Outputar) == 0 {
			return Char{}
		}
		parts := strings.Split(world.Outputar[world.Currentturn]," ")
		cname := strings.Join(parts[0:len(parts)-1], " ")
		//fmt.Println("Current name: ", cname)
		if len(parts) >= 2 {
			//fmt.Println("Ugh...", cname)

			preferPosHp := Char{}
			for k := range world.Npcs {
				if world.Npcs[k].Name == cname {
					if world.Npcs[k].CurHP <= 0 {
						preferPosHp = world.Npcs[k]
					} else {
						return world.Npcs[k]
					}
				}
			}

			if preferPosHp.Name != "" {
				return preferPosHp
			}

			for k := range world.Chars {
				if world.Chars[k].Name == cname && world.Chars[k].InParty {
					return world.Chars[k]
				}
			}

			sendConsole(fmt.Sprintln("Failed to find npc or party char with turn..."))


		}
		sendConsole(fmt.Sprintln("Failed to lookup anything with ", world.Outputar[world.Currentturn]))
	//}

	return world.Npcs[0]

}

func getRandomNpc() Char {
	if len(world.Npcs) == 1 {
		return world.Npcs[0]
	}

	for {
		dres := getDiceResults(fmt.Sprintf("1d%d",len(world.Npcs)))
		dres = strings.TrimRight(dres, " \n")
		ran, _ := strconv.Atoi(dres)
		if len(world.Npcs) == 0 {
			return Char{}
		}
		if world.Npcs[ran-1].CurHP > 0 {
			return world.Npcs[ran-1]
		}
	}
}

func getRandomPartyCharOrOtherRace(race string) Char {
	// haha this is crazy
	for {
		dres1 := getDiceResults("1d10")
		dres1 = strings.TrimRight(dres1, " \n")
		ran1, _ := strconv.Atoi(dres1)


		attacknonrace := false
		if ran1 < 5 {
			attacknonrace = true
		}

		dres := getDiceResults(fmt.Sprintf("1d%d",len(world.Players)))
		dres = strings.TrimRight(dres, " \n")
		ran, _ := strconv.Atoi(dres)

		for i := range world.Chars {
			if attacknonrace && world.Chars[i].Race != race && charIsNpc(world.Chars[i].Key) {
				fmt.Println("ATTACKING NON RACE CHAR!",world.Chars[i].Key, charIsNpc(world.Chars[i].Key))
				return world.Chars[i]
			} else if world.Chars[i].Playername == world.Players[ran-1] && getHP(world.Chars[i].Key) > 0 {
				return world.Chars[i]
			}
		}
	}


}

func getRandomPartyChar() Char {


	// haha this is crazy
	for {
		dres := getDiceResults(fmt.Sprintf("1d%d",len(world.Players)))
		dres = strings.TrimRight(dres, " \n")
		ran, _ := strconv.Atoi(dres)

		for i := range world.Chars {
			if world.Chars[i].Playername == world.Players[ran-1] && getHP(world.Chars[i].Key) > 0 {
				return world.Chars[i]
			}
		}
	}


	return Char{}
}

func getNumNpcInstances(charname string) int {
	count := 0
	for i := range world.Npcs {
		if world.Npcs[i].Name == charname && world.Npcs[i].CurHP > 0 {
			count++
		}
	}

	return count
}

func allpdead() bool {
	for i := range world.Players {
		for k := range world.Chars {
			if world.Chars[k].Playername == world.Players[i] && getHP(world.Chars[k].Key) > 0 {
				sendConsole(fmt.Sprintln(world.Chars[k].Key, " is not dead yet."))
				return false
			}
		}
	}

	return true
}

func allndead() bool {
	for i := range world.Npcs {
		if world.Npcs[i].CurHP > 0 {
			return false
		}
	}

	return true

}


func autoFight() string {
	for {
		msg := ""
		cmd := &Command{}

		cchar := getCharWithTurn()
		if !cchar.InParty {
			numNpcs := getNumNpcInstances(cchar.Name)
			sendConsole(fmt.Sprintln("Num instances for", cchar.Name, " is ", numNpcs))

			AT: for i := 0; i < numNpcs; i++ {
				cmd = parseInput("deriv","ant");
				msg = autoAttack()
				world.Lastoutput = renderContent(msg, cmd)
				h.broadcast <- []byte(world.Lastoutput)
				time.Sleep(2000 * time.Millisecond)

				if allpdead() {
					break AT
				}
			}
		} else {
			cmd = parseInput("deriv","ant");
			msg = autoAttack()
			world.Lastoutput = renderContent(msg, cmd)
			h.broadcast <- []byte(world.Lastoutput)
			time.Sleep(2000 * time.Millisecond)
		}


		msg = ""
		nextTurn()
		cmd = parseInput("deriv","nt");
		world.Lastoutput = renderContent(msg, cmd)
		h.broadcast <- []byte(world.Lastoutput)
		time.Sleep(100 * time.Millisecond)


		if len(world.Outputar) == 0 {
			sendConsole("Ending combat.")
			return ""
		}

		if allpdead() {
			sendConsole(fmt.Sprintln("All players dead, exiting autof."))
			return ""
		}



		if allndead() {
			sendConsole(fmt.Sprintln("All world.Npcs dead, exiting autof."))
			return ""
		}

	}
}

func autoAttack() string {
	msg := ""
	cchar := getCharWithTurn()
	hp := getHP(cchar.Key)
	if cchar.InParty && hp > 0 {
		randomNpc := getRandomNpc()
		if randomNpc.CurHP <= 0 {
			sendConsole(fmt.Sprintln("Target npc is dead!", randomNpc.CurHP, randomNpc.Key))
		} else {
			msg = attack(cchar, 0, randomNpc, "")
			//nextTurn()
		}
	} else if cchar.InParty && hp < 0 {
		sendConsole(fmt.Sprintln("Source is dead!"))
		msg = cchar.Name + " is dead."
		//nextTurn()
	} else if cchar.CurHP >= 0 && charIsNpc(cchar.Key) {
		randomChar := getRandomPartyCharOrOtherRace(cchar.Race)
		thp := getHP(randomChar.Key)
		if thp <= 0 {
			sendConsole(fmt.Sprintln("Target player/npc is dead!", thp, randomChar.Name))
		} else {
			msg = attack(cchar, 0, randomChar, "")
			//nextTurn()
		}
	} else {
		sendConsole(fmt.Sprintln("Source is dead!"))
		msg = cchar.Name + " is dead."
		//nextTurn()
	}

	return msg
}

func executeCommand(cmd Command) string {
	msg := ""

		if (cmd.Name == "roll" || cmd.Name == "r") && len(cmd.Args) >= 1 {
			diceresults := getDiceResults(cmd.Args[0])
			if len(cmd.Args) == 2 {
				diceresults2 := getDiceResults(cmd.Args[0])
				msg = fmt.Sprintf("Roll 1 %s = %s", cmd.Args[0], diceresults)
				msg = msg + "<br>" + fmt.Sprintf("Roll 1 %s = %s", cmd.Args[0], diceresults2)
			} else {
				msg = fmt.Sprintf("Roll %s = %s", cmd.Args[0], diceresults)
			}
		} else if cmd.Name == "c" {
			msg = " "
		} else if cmd.Name == "help" {
			sendConsole("stat - show overall status, place and NPC health\nls [places|chars|npcs] - list all objects of a particular type\nplace PLACE - (p) change to PLACE\ndrop NAME - drop an instance of NAME into the place. This will be an NPC and NAME will be the key from the 'ls chars' list.\ncombat - enter combat rounds and roll initiative\nendcombat - ends combat rounds and removes initiative\natt NAME.ATTINDEX TARGET - attack TARGET NPC or player with by NAME and use attack type (0 - n) specified by ATTINDEX\nnt - advance to next turn in initiative ranking\npt - return to previous turn in initiative ranking\nreset - reset all state\nclearnpcs - clears out NPCS\nreload - reloads all configuration data\nsethp CHAR - sets HP of kCHAR\nsubhp CHAR - subtract HP from CHAR\naddhp CHAR - add HP to CHAR\nroll DICESTRING - (r) roll a dice string (e.g., 1d4+2) and show it on the main page\nrq - roll a dice string but only print to console\nv - view a character\nmsg - send an arbitrary message to the players\nclear - (c) clear any message or output\n")
			msg = " "
		} else if cmd.Name == "autof" {
			go autoFight()
			msg = " "
		} else if cmd.Name == "ant" {
			autoAttack()
		} else if cmd.Name == "nt" {
			nextTurn()
			msg = " "
		} else if cmd.Name == "pt" {
			prevTurn()
			msg = " "
		} else if cmd.Name == "stat"  || cmd.Name == "status" {
			printStatus()
		} else if cmd.Name == "blog" {
			msg = world.Battlelog
		} else if (cmd.Name == "rollq" || cmd.Name == "rq") && len(cmd.Args) >= 1 {
			diceresults := getDiceResults(cmd.Args[0])
			sendConsole(fmt.Sprintf("Roll %s = %s\n", cmd.Args[0], diceresults))
		} else if cmd.Name == "msg" {
			msg = fmt.Sprintf("%s", cmd.RawArgs)
		} else if cmd.Name == "cmsg" {
			msg = ""
			sendConsole(fmt.Sprintln(cmd.RawArgs))
		} else if cmd.Name == "drop" {
			dropNpc(cmd.Args[0])
			msg = " "
		} else if cmd.Name == "dropran" {
			lnc := len(world.Chars)
			dr := getDiceResults(fmt.Sprintf("1d%d",lnc))
			//FIXME doesn't work with local dice implementation
			dri,_ := strconv.Atoi(dr)
			if world.Chars[dri].InParty {
				sendConsole(fmt.Sprintln("Oops, chose a player on random drop."))
				msg = ""
			} else {
				dropNpc(world.Chars[dri].Name)
				msg = " "
			}
		} else if len(cmd.Args) >= 1 && (cmd.Name == "place" || cmd.Name == "p") {
			initNpcs()
			world.Place = cmd.Args[0]
			pl := getPlace(world.Place)
			for i := range pl.Autodrop {
				dropNpc(pl.Autodrop[i])
			}
			world.Music = pl.Music
			msg = " "
		} else if cmd.Name == "vo" && len(cmd.Args) >= 1 {
			msg = viewObj(cmd.Args[0])
		} else if cmd.Name == "cvo" && len(cmd.Args) >= 1 {
			msg = " "
			cmsg := viewObjConsole(cmd.Args[0])
			sendConsole(cmsg)
		} else if cmd.Name == "v" && len(cmd.Args) >= 1 {
			if charIsNpc(cmd.Args[0]) {
				msg = viewChar(cmd.Args[0],false)
			} else {
				msg = viewChar(cmd.Args[0],true)
			}
		} else if cmd.Name == "cv" && len(cmd.Args) >= 1 {
			msg = " "
			cmsg := viewCharConsole(cmd.Args[0])
			sendConsole(cmsg)
		} else if cmd.Name == "sethp" {
			aint, _ := strconv.Atoi(cmd.Args[1])
			setHP(cmd.Args[0], aint)
			msg = " "
		} else if cmd.Name == "subhp" && len(cmd.Args) >= 2 {
			aint, _ := strconv.Atoi(cmd.Args[1])
			hp := getHP(cmd.Args[0])
			setHP(cmd.Args[0], hp-aint)
			msg = " "
		} else if cmd.Name == "addhp" && len(cmd.Args) >= 2 {
			aint, _ := strconv.Atoi(cmd.Args[1])
			hp := getHP(cmd.Args[0])
			setHP(cmd.Args[0], hp+aint)
			msg = " "
		} else if cmd.Name == "t" {
			if world.NoText {
				world.NoText = false
			} else {
				world.NoText = true
			}
			msg = " "
		} else if cmd.Name == "music" {
			world.Music = cmd.Args[0]
			msg = fmt.Sprintf("Music set to %s",cmd.Args[0])
		} else if cmd.Name == "sp" {
			if world.ShowParty {
				world.ShowParty = false
			} else {
				world.ShowParty = true
			}
			msg = " "
		} else if cmd.Name == "snp" {
			if world.ShowNpcs {
				world.ShowNpcs = false
			} else {
				world.ShowNpcs = true
			}
			msg = " "
		} else if cmd.Name == "smugs" {
			if world.ShowMugs {
				world.ShowMugs = false
			} else {
				world.ShowMugs = true
			}
			msg = " "
		} else if cmd.Name == "ls" {
			lsarg := ""
			if len(cmd.Args) == 2 {
				lsarg = cmd.Args[1]
			}

			if len(cmd.Args) >= 1 && cmd.Args[0] == "places" {
				listPlaces(lsarg)
			} else if len(cmd.Args) >= 1 && cmd.Args[0] == "chars" {
				listChars(lsarg)
			} else if len(cmd.Args) >= 1 && cmd.Args[0] == "objs" {
				listObjs(lsarg)
			} else {
				listNpcs()
			}
		} else if cmd.Name == "clearnpcs" {
			initNpcs()
			msg = " "
		} else if cmd.Name == "combat" {
			world.Initiativetxt = rollInitiatives(cmd.RawArgs)
			world.Music = "fight_real.ogg"
			msg = " "
		} else if cmd.Name == "reset" {
			initialState(&world)
			msg = " "
		} else if cmd.Name == "endcombat" {
			world.Initiativetxt = ""
			world.Outputar = make([]string, 0)
			world.Battlelog = ""
			world.Currentturn = 0
			world.Music = "Off"
			msg = " "
		} else if cmd.Name == "att" && len(cmd.Args) > 1 && strings.Contains(cmd.Args[0], ".") {
			a := strings.Split(cmd.Args[0], ".")
			atti, _ := strconv.Atoi(a[1])
			adv := ""
			if len(cmd.Args) == 3 {
				adv = cmd.Args[2]
			}
			char1 := getChar(a[0])
			char2 := getChar(cmd.Args[1])

			if char1.Name != "" && char2.Name != "" {
				msg = attack(char1, atti, char2, adv)
			} else if (char1.Name == "") {
				sendConsole(fmt.Sprintln("Failed to load ", a[0]))
			} else if (char2.Name == "") {
				sendConsole(fmt.Sprintln("Failed to load ", cmd.Args[1]))
			}
		} else if cmd.Name == "re" || cmd.Name == "reload" {
			sendConsole(fmt.Sprintln("Reload Configuration"))
			initPlaces()
			initChars(true)
			initObjects()
			syncNpcs()
			msg = " "
		}



	fmt.Println(msg)
	return msg
}

func loopForDMInput() {
	consolereader := bufio.NewReader(os.Stdin)
	for {

		input, _ := consolereader.ReadString('\n')
		cmd := parseInput("console",input)
		//msg := ""
		//fmt.Println("Running command: ", cmd.Name)

		msg := executeCommand(*cmd)

		//sendConsole(fmt.Sprintf("> "))
		fmt.Printf("> ")
		if msg != "" {
			world.Lastoutput = renderContent(msg, cmd)
		}
		h.broadcast <- []byte(world.Lastoutput)
	}
}

func loadPlayerChar(playername string) Char {
	filename := fmt.Sprintf("assets/players/%s.json",playername)
	file, err := os.Open(filename)

	if err != nil {
		panic(fmt.Sprintf("Could not open %s",filename))
	}

	filebytes := ReadFileContents(file)

	char := Char{}
	err = json.Unmarshal(filebytes, &char)
	if err != nil {
		fmt.Println("Failed to read player file: ", err)
		panic(err)
	}

	return char

}

func updatePlayerChar(req *http.Request) {
	char := Char{}

	char.Name = req.Form["charname"][0]
	char.Class = req.Form["class"][0]
	char.Race = req.Form["race"][0]
	char.Level,_ = strconv.Atoi(req.Form["level"][0])
	char.Alignment = req.Form["alignment"][0]
	abil := Abilities{}
	json.Unmarshal([]byte(req.Form["abilities"][0]), &abil)
	char.Abilities = abil
	//char.Abilities = req.Form["abilities"]
	char.HP,_ = strconv.Atoi(req.Form["hp"][0])
	char.AC,_ = strconv.Atoi(req.Form["ac"][0])
	char.Initiative,_ = strconv.Atoi(req.Form["initiative"][0])
	//char.Attacks,_ = req.Form["attacks"]
	atts := make([]Attack,0)
	json.Unmarshal([]byte(req.Form["attacks"][0]), &atts)
	char.Attacks = atts
	//char.Inventory,_ = req.Form["inventory"]
	inv := make([]string,0)
	//fmt.Println(inv)
	json.Unmarshal([]byte(req.Form["inventory"][0]), &inv)
	char.Inventory = inv
	char.Background = req.Form["background"][0]
	char.Image = req.Form["image"][0]
	char.Inspiration,_ = strconv.Atoi(req.Form["inspiration"][0])
	char.ProfBonus,_ = strconv.Atoi(req.Form["profbonus"][0])
	char.PassPerception,_ = strconv.Atoi(req.Form["passperception"][0])
	char.HitDice = req.Form["hitdice"][0]
	char.Speed,_ = strconv.Atoi(req.Form["speed"][0])
	char.Skills = req.Form["skills"][0]
	char.MiscProfLanguages = req.Form["misclang"][0]
	char.PersonalityTraits = req.Form["personalitytraits"][0]
	char.Ideals = req.Form["ideals"][0]
	char.Bonds = req.Form["bonds"][0]
	char.Flaws = req.Form["flaws"][0]
	char.FeaturesTraits = req.Form["featurestraits"][0]
	char.Treasure = req.Form["treasure"][0]

	char.SpellL0 = req.Form["spelll0"][0]
	char.SpellL1 = req.Form["spelll1"][0]
	char.SpellL2 = req.Form["spelll2"][0]
	char.SpellL3 = req.Form["spelll3"][0]
	char.SpellL4 = req.Form["spelll4"][0]
	char.SpellL5 = req.Form["spelll5"][0]
	char.SpellL6 = req.Form["spelll6"][0]
	char.SpellL7 = req.Form["spelll7"][0]

	if req.Form["inparty"][0] == "true" {
		char.InParty = true
	} else {
		char.InParty = false
	}
	char.Playername = req.Form["playername"][0]


	//fmt.Println(char)
	charbytes,_ := json.Marshal(char)
	//fmt.Println(string(bytes))

	buf := bytes.NewBuffer(make([]byte,0))
	json.Indent(buf,charbytes,"","  ")
	//fmt.Println(string(buf.Bytes()))
	fmt.Println(req.Form["playername"][0], "updated!")

	updatePlayerFile(req.Form["playername"][0],buf.Bytes())

}

func updatePlayerFile(charname string, data []byte) {


	if charname == "" {
		fmt.Println("Error empty playername passed to updatePlayerFile()\n")
	}
	filename := fmt.Sprintf("assets/players/%s.json",charname)

	err := ioutil.WriteFile(filename,data,0755)

	if err != nil {
		panic(fmt.Sprintf("Could not write to %s", filename, " ", err))
	}
	initPlaces()
	initChars(false)
	initObjects()
	syncNpcs()
	world.Lastoutput = renderContent("", &Command{})
}

func editPlayerChar(c http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	playercookie,_ := req.Cookie("playername")
	fmt.Println("Player cookie: ", playercookie.Value)
	if len(req.Form["update"]) != 0 {
		// update player data
		updatePlayerChar(req)
		http.Redirect(c,req,"/",302)
		//mainData := MainData{Host: req.Host, Content: "Successfully updated the char!<br><a href=\"/\">Return to main view</a><br>"}
		//homeTempl.Execute(c, mainData)
	} else if playercookie.Value != "" {
		// send form
		//mainData := MainData{Content: editPlayerForm(req)}
		char := loadPlayerChar(strings.ToLower(playercookie.Value))
		fmt.Println("Loaded ", char.Name)
		char.JsonIfy()
		editPlayerTempl.Execute(c, char)
	}
}

func issueAttack(c http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	char1 := getChar(req.Form["char"][0])
	char2 := getChar(req.Form["target"][0])

	if world.Currentturn == 0 {
		fmt.Println("Attack out of combat!")
		http.Redirect(c,req,"/",302)
		return
	}

	cchar := getCharWithTurn()
	if cchar.Name != char1.Name {
		fmt.Println("Attack out of order!")
		http.Redirect(c,req,"/",302)
		return
		//mainData := MainData{Host: req.Host, Content: lastoutput}
		//homeTempl.Execute(c, mainData)
		//return
	}

	if char1.Name != "" && char2.Name != "" {
		atti,_ := strconv.Atoi(req.Form["attack"][0])
		msg := attack(char1, atti, char2, "")
		cmd := parseInput("deriv",fmt.Sprintf("att %s.%d %s", char1.Key, atti, char2.Key))
		//cmd := Command{Name: "att", RawArgs: fmt.Sprintf("%s.%d %s", char1.Key, atti, char2.Key);
		world.Lastoutput = renderContent(msg, cmd)
		h.broadcast <- []byte(world.Lastoutput)
	} else if (char1.Name == "") {
		fmt.Println("Failed to load ", req.Form["char"][0])
	} else if (char2.Name == "") {
		fmt.Println("Failed to load ", req.Form["char"][0])
	}

	mainData := MainData{Host: req.Host, Content: world.Lastoutput}
	homeTempl.Execute(c, mainData)
}

func webViewChar(c http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	playercookie,_ := req.Cookie("playername")
	if playercookie != nil {
		fmt.Println("Player cookie: ", playercookie.Value)
	}

	if len(req.Form["name"]) == 0 {
		mainData := MainData{Host: req.Host, Content: world.Lastoutput}
		homeTempl.Execute(c, mainData)
	} else {
		char := getChar(req.Form["name"][0])
		// does the same thing now but may provide permissions to view in the future
		if playercookie != nil && char.Playername == playercookie.Value {
			mainData := MainData{Host: req.Host, Content: viewChar(req.Form["name"][0],true)}
			homeTempl.Execute(c, mainData)
		} else {
			mainData := MainData{Host: req.Host, Content: viewChar(req.Form["name"][0],true)}
			homeTempl.Execute(c, mainData)
		}
	}
}

var chttp = http.NewServeMux()

func initialState(world *WorldState) {
	world.Players = strings.Split(world.Charlist,",")
	world.Curhps = make(map[int]int)
	initPlaces()
	initObjects()
	initChars(true)
	initNpcs()
	world.NoText = false
	world.ShowParty = true
	world.ShowMugs = true
	world.ShowNpcs = true
	world.Place = "void"
	world.Initiativetxt = ""
	world.Outputar = make([]string, 0)
	world.Battlelog = ""

	world.Abilitymods = map[int]int {
		1: -5,
		2: -4,
		3: -4,
		4: -3,
		5: -3,
		6: -2,
		7: -2,
		8: -1,
		9: -1,
		10: 0,
		11: 0,
		12: 1,
		13: 1,
		14: 2,
		15: 2,
		16: 3,
		17: 3,
		18: 4,
		19: 4,
		20: 5,
		21: 5,
		22: 6,
		23: 6,
		24: 7,
		25: 7,
		26: 8,
		27: 8,
		28: 9,
		29: 9,
		30: 10,
	}


	world.Challtable = map[int]int {
		0: 100,
		1: 200,
		2: 450,
		3: 700,
		4: 1100,
		5: 1800,
		6: 2300,
		7: 2900,
		8: 3900,
		9: 5000,
		10: 5900,
		11: 7200,
		12: 8400,
		13: 10000,
		14: 11500,
		15: 13000,
		16: 15000,
		17: 18000,
		18: 20000,
		19: 22000,
		20: 25000,
		21: 33000,
		22: 41000,
		23: 50000,
		24: 62000,
		25: 75000,
		26: 90000,
		27: 105000,
		28: 120000,
		29: 135000,
		30: 155000,
	}

	world.Exptable = map[int]int {
		1: 0,
		2: 300,
		3: 900,
		4: 2700,
		5: 6500,
		6: 14000,
		7: 23000,
		8: 34000,
		9: 48000,
		10: 64000,
		11: 85000,
		12: 100000,
		13: 120000,
		14: 140000,
		15: 165000,
		16: 195000,
		17: 225000,
		18: 265000,
		19: 305000,
	}


}

func handle_telnet(conn net.Conn) {
	telnetreader := bufio.NewReader(conn)

	telconn := telnetconn{conn: &conn, send: make(chan []byte, 256)}
	h.telregis <- &telconn
	fmt.Println("REGISTERING", conn.RemoteAddr())
	go telconn.writer()
	defer func() { h.telunregis <- &telconn }()

	magicstring := "ohgodmedusa"

	count := 0
	for {

		line, err := telnetreader.ReadString('\n')
		// getting some weird data here, I assume a \r not sure, this is at stupid hack
		// that works, but FIXME
		if len(line) <= 1 {
			return
		}
		line = line[:len(line)-2]
		if err != nil { return }

		if count == 0 && !strings.Contains(line,magicstring) {
			fmt.Println("FAILED SECRET WORD", conn.RemoteAddr())
			break
		}
		count++

		cmd := parseInput(conn.RemoteAddr().String(),line)
		//fmt.Println(cmd)
		msg := executeCommand(*cmd)

		//h.telbroadcast <- []byte("> ")
		telconn.send <- []byte("> ")

		if msg != "" {
			world.Lastoutput = renderContent(msg, cmd)
		}
		h.broadcast <- []byte(world.Lastoutput)


	}

	fmt.Println("UNREGISTERING")
	h.telunregis <- &telconn
}

func spawnTelnetService() {
	psock, err := net.Listen("tcp", ":9000")
	if err != nil { return }

	for {
		conn, err := psock.Accept()
		if err != nil { return }

		go handle_telnet(conn)
	}
}

func sendConsole(txt string) {
	txt = strings.Replace(txt,"\n","\n\r",-1)
	fmt.Printf(txt)
	h.telbroadcast <- []byte(txt)
}

func main() {
	flag.StringVar(&world.Charlist, "chars", "", "Character list separate by commas.")
	flag.Parse()

	homeTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "home.html")))
	editPlayerTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "editplayer.html")))
	playerIdTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "playerid.html")))

	//world := WorldState{}
	initialState(&world)

	world.Lastoutput = renderContent("", &Command{})

	world.Loggedin = make([]string,0)
	go h.run()
	go loopForDMInput()
	go spawnTelnetService()

	chttp.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/ws", wsHandler)


	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}
