package main

import (
	"encoding/json"

	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"
	rpch "github.com/tidusant/chadmin-repo/cuahang"
	"github.com/tidusant/chadmin-repo/models"
	//repush
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"strconv"
	"strings"
)

const (
	defaultcampaigncode string = "XVsdAZGVmY"
)

type Arith int

func (t *Arith) Run(data string, result *string) error {
	log.Debugf("Call RPC orders args:" + data)
	*result = ""
	//parse args
	args := strings.Split(data, "|")

	if len(args) < 3 {
		return nil
	}
	var usex models.UserSession
	usex.Session = args[0]
	usex.Action = args[2]
	info := strings.Split(args[1], "[+]")
	usex.UserID = info[0]
	ShopID := info[1]
	usex.Params = ""
	if len(args) > 3 {
		usex.Params = args[3]
	}

	//check shop permission
	shop := rpch.GetShopById(usex.UserID, ShopID)
	if shop.Status == 0 {
		*result = c3mcommon.ReturnJsonMessage("-4", "Shop is disabled.", "", "")
		return nil
	}
	usex.Shop = shop

	if usex.Action == "la" {
		*result = LoadAll(usex)
	} else if usex.Action == "s" {
		*result = SaveShipper(usex)
	} else if usex.Action == "rm" {
		*result = DeleteShipper(usex)
	} else { //default
		*result = c3mcommon.ReturnJsonMessage("-5", "Action not found.", "", "")
	}

	return nil
}

func SaveShipper(usex models.UserSession) string {

	var shipper models.Shipper
	err := json.Unmarshal([]byte(usex.Params), &shipper)
	if !c3mcommon.CheckError("update shipper parse json", err) {
		return c3mcommon.ReturnJsonMessage("0", "update shipper fail", "", "")
	}
	//check old status
	oldshipper := shipper
	if oldshipper.ID.Hex() != "" {
		oldshipper = rpch.GetShipperByID(shipper.ID.Hex(), usex.Shop.ID.Hex())
		oldshipper.Name = shipper.Name
		oldshipper.Default = shipper.Default
		oldshipper.Color = shipper.Color
	} else {
		oldshipper.UserId = usex.UserID
		oldshipper.ShopId = usex.Shop.ID.Hex()
	}

	//check default
	if oldshipper.Default == true {
		rpch.UnSetShipperDefault(usex.Shop.ID.Hex())
	}
	if oldshipper.Color == "" {
		oldshipper.Color = "ffffff"
	}

	oldshipper = rpch.SaveShipper(oldshipper)
	b, _ := json.Marshal(oldshipper)
	return c3mcommon.ReturnJsonMessage("1", "", "success", string(b))
}

func DeleteShipper(usex models.UserSession) string {
	//get stat
	shipper := rpch.GetShipperByID(usex.Params, usex.Shop.ID.Hex())
	if shipper.ID.Hex() == "" {
		return c3mcommon.ReturnJsonMessage("-5", "shipper not found.", "", "")
	}
	if shipper.Default {
		return c3mcommon.ReturnJsonMessage("-5", "shipper is default. Please select another shipper default.", "", "")
	}
	//check status empty
	count := rpch.GetCountOrderByShipper(shipper)
	//check old status
	if count > 0 {
		return c3mcommon.ReturnJsonMessage("-5", "shipper not empty. "+strconv.Itoa(count)+" shipper use this status", "", "")
	}

	rpch.DeleteShipper(shipper)

	return c3mcommon.ReturnJsonMessage("1", "", "success", "")
}

func LoadAll(usex models.UserSession) string {

	//default status
	camps := rpch.GetAllShipper(usex.Shop.ID.Hex())

	info, _ := json.Marshal(camps)
	strrt := string(info)
	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)
}

func main() {
	var port int
	var debug bool
	flag.IntVar(&port, "port", 9886, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	logLevel := log.DebugLevel
	if !debug {
		logLevel = log.InfoLevel

	}

	log.SetOutputFile(fmt.Sprintf("adminShipper-"+strconv.Itoa(port)), logLevel)
	defer log.CloseOutputFile()
	log.RedirectStdOut()

	//init db
	arith := new(Arith)
	rpc.Register(arith)
	log.Infof("running with port:" + strconv.Itoa(port))

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	c3mcommon.CheckError("rpc dail:", err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	c3mcommon.CheckError("rpc init listen", err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}
