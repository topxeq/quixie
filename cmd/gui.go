//go:build !linux && !darwin
// +build !linux,!darwin

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ncruces/zenity"
	"github.com/topxeq/dlgs"
	"github.com/topxeq/quixie"

	// "github.com/topxeq/go-sciter"
	// "github.com/topxeq/go-sciter/window"
	"github.com/sciter-sdk/go-sciter"
	"github.com/sciter-sdk/go-sciter/window"
	"github.com/topxeq/tk"

	"github.com/kbinani/screenshot"

	"github.com/jchv/go-webview2"
)

// ...
// "github.com/jchv/go-webview2"
// "github.com/jchv/go-webview2/pkg/edge"
// )

// func main() {
// dataPath, _ := filepath.Abs("./userdata")
// w := webview2.NewWithOptions(webview2.WebViewOptions{
// 	Debug:     true,
// 	AutoFocus: true,
// 	DataPath:  dataPath,
// 	WindowOptions: webview2.WindowOptions{
// 		Title: "go-webview2 Example",
// 	},
// })
// if w == nil {
// 	log.Fatalln("Failed to load webview.")
// }
// defer w.Destroy()

// // update window icon
// w32.SendMessage(w.Window(), 0x0080, 1, w32.ExtractIcon(os.Args[0], 0))

// w.SetSize(800, 600, webview2.HintNone)

// chromium := getChromium(w)

// folderPath, _ := filepath.Abs("./public")
// webview := chromium.GetICoreWebView2_3()
// webview.SetVirtualHostNameToFolderMapping(
// 	"app.assets", folderPath,
// 	edge.COREWEBVIEW2_HOST_RESOURCE_ACCESS_KIND_DENY_CORS,
// )
// w.Navigate("http://app.assets/index.html")

// w.Run()
// }

// func getChromium(w webview2.WebView) *edge.Chromium {
// browser := reflect.ValueOf(w).Elem().FieldByName("browser")
// browser = reflect.NewAt(browser.Type(), unsafe.Pointer(browser.UnsafeAddr())).Elem()
// return browser.Interface().(*edge.Chromium)
// }

func newWindowWebView2(objA interface{}, paramsA []interface{}) interface{} {
	var paraArgsT []string = []string{}

	for i := 0; i < len(paramsA); i++ {
		paraArgsT = append(paraArgsT, tk.ToStr(paramsA[i]))
	}

	p := objA.(*quixie.QuixieVM)

	titleT := p.GetSwitchVarValue(p.Running, paraArgsT, "-title=", "dialog")
	widthT := p.GetSwitchVarValue(p.Running, paraArgsT, "-width=", "800")
	heightT := p.GetSwitchVarValue(p.Running, paraArgsT, "-height=", "600")
	iconT := p.GetSwitchVarValue(p.Running, paraArgsT, "-icon=", "2")
	debugT := tk.IfSwitchExistsWhole(paraArgsT, "-debug")
	centerT := tk.IfSwitchExistsWhole(paraArgsT, "-center")
	fixT := tk.IfSwitchExistsWhole(paraArgsT, "-fix")
	maxT := tk.IfSwitchExistsWhole(paraArgsT, "-max")
	minT := tk.IfSwitchExistsWhole(paraArgsT, "-min")

	if maxT {
		// windowStyleT = webview2.HintMax

		rectT := screenshot.GetDisplayBounds(0)

		widthT = tk.ToStr(rectT.Max.X)
		heightT = tk.ToStr(rectT.Max.Y)
	}

	if minT {
		// windowStyleT = webview2.HintMin

		widthT = "0"
		heightT = "0"
	}

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     debugT,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  titleT,
			Width:  uint(tk.ToInt(widthT, 800)),
			Height: uint(tk.ToInt(heightT, 600)),
			IconId: uint(tk.ToInt(iconT, 2)), // icon resource id
			Center: centerT,
		},
	})

	if w == nil {
		return fmt.Errorf("?????????????????????%v", "N/A")
	}

	windowStyleT := webview2.HintNone

	if fixT {
		windowStyleT = webview2.HintFixed
	}

	w.SetSize(tk.ToInt(widthT, 800), tk.ToInt(heightT, 600), windowStyleT)

	var handlerT tk.TXDelegate

	handlerT = func(actionA string, objA interface{}, dataA interface{}, paramsA ...interface{}) interface{} {
		switch actionA {
		case "show":
			w.Run()
			return nil
		case "navigate":
			len1T := len(paramsA)
			if len1T < 1 {
				return fmt.Errorf("????????????")
			}

			if len1T > 0 {
				w.Navigate(tk.ToStr(paramsA[0]))
			}

			return nil
		case "setHtml":
			len1T := len(paramsA)
			if len1T < 1 {
				return fmt.Errorf("????????????")
			}

			if len1T > 0 {
				w.SetHtml(tk.ToStr(paramsA[0]))
			}

			return nil
		case "call", "eval":
			len1T := len(paramsA)
			if len1T < 1 {
				return fmt.Errorf("????????????")
			}

			if len1T > 0 {
				w.Dispatch(func() {
					w.Eval(tk.ToStr(paramsA[0]))
				})
			}

			return nil
		case "close":
			w.Destroy()
			return nil
		case "setQuickDelegate":
			var deleT tk.QuickVarDelegate = paramsA[0].(tk.QuickVarDelegate)

			w.Bind("quickDelegateDo", func(args ...interface{}) interface{} {
				// args???WebView2??????????????????????????????????????????
				// ?????????????????????????????????????????????????????????
				// strT := args[0].String()

				rsT := deleT(args...)

				// ??????????????????????????????????????????????????????
				return rsT
			})

			return nil
		case "setDelegate":
			var codeT string = tk.ToStr(paramsA[0])

			codeT = strings.ReplaceAll(codeT, "~~~", "`")

			// p := objA.(*quixie.QuixieVM)

			w.Bind("delegateDo", func(argsA ...interface{}) interface{} {
				// args???WebView2??????????????????????????????????????????
				// ?????????????????????????????????????????????????????????
				// strT := args[0].String()

				vmT := quixie.NewVMQuick()

				vmT.SetVar(p.Running, "inputG", argsA)

				lrs := vmT.Load(vmT.Running, codeT)

				if tk.IsErrX(lrs) {
					return lrs
				}

				rs := vmT.Run()

				// if !tk.IsErrX(rs) {
				// 	outIndexT, ok := vmT.VarIndexMapM["outG"]
				// 	if !ok {
				// 		return tk.ErrStrf("no result")
				// 	}

				// 	return tk.ToStr((*vmT.FuncContextM.VarsM)[vmT.FuncContextM.VarsLocalMapM[outIndexT]])
				// }

				// ??????????????????????????????????????????????????????
				return rs
			})

			return nil
		case "setGoDelegate":
			var codeT string = tk.ToStr(paramsA[0])

			// p := objA.(*quixie.QuixieVM)

			w.Bind("goDelegateDo", func(args ...interface{}) interface{} {
				// args???WebView2??????????????????????????????????????????
				// ?????????????????????????????????????????????????????????
				// strT := args[0].String()

				vmT := quixie.NewVMQuick()

				// quixie.GlobalsG.Vars["verbose"]. = p.VerboseM
				// vmT.VerbosePlusM = p.VerbosePlusM

				vmT.SetVar(vmT.Running, "inputG", args)

				// argCountT := p.Pop()

				// if argCountT == Undefined {
				// 	return tk.ErrStrf()
				// }

				// for i := 0; i < argCountA; i++ {
				// 	vmT.Push(p.Pop())
				// }

				lrs := vmT.Load(vmT.Running, codeT)

				if tk.IsErrX(lrs) {
					return lrs
				}

				go vmT.Run()

				// ??????????????????????????????????????????????????????
				return ""
			})

			return nil
		// case "call":
		// 	len1T := len(paramsA)
		// 	if len1T < 1 {
		// 		return fmt.Errorf("????????????")
		// 	}

		// 	if len1T > 1 {
		// 		aryT := make([]*sciter.Value, 0, 10)

		// 		for i := 1; i < len1T; i++ {
		// 			aryT = append(aryT, sciter.NewValue(paramsA[i]))
		// 		}

		// 		rsT, errT := w.Call(tk.ToStr(paramsA[0]), aryT...)

		// 		if errT != nil {
		// 			return fmt.Errorf("??????????????????????????????%v", errT)
		// 		}

		// 		return rsT.String()
		// 	}

		// 	rsT, errT := w.Call(tk.ToStr(paramsA[0]))

		// 	if errT != nil {
		// 		return fmt.Errorf("??????????????????????????????%v", errT)
		// 	}

		// 	return rsT.String()
		default:
			return fmt.Errorf("???????????????%v", actionA)
		}

		return nil
	}

	// w.Show()
	// w.Run()

	return handlerT

}

func guiHandler(actionA string, objA interface{}, dataA interface{}, paramsA ...interface{}) interface{} {
	switch actionA {
	case "init":
		rs := initGUI()
		return rs
	case "lockOSThread":
		runtime.LockOSThread()
		return nil
	case "method", "mt":
		if len(paramsA) < 2 {
			return fmt.Errorf("????????????")
		}

		objT := paramsA[0]

		methodNameT := tk.ToStr(paramsA[1])

		v1p := 2

		switch nv := objT.(type) {
		case zenity.ProgressDialog:
			switch methodNameT {
			case "close":
				rs := nv.Close()
				return rs
			case "complete":
				rs := nv.Complete()
				return rs
			case "text":
				if len(paramsA) < v1p+1 {
					return fmt.Errorf("????????????")
				}

				v1 := tk.ToStr(paramsA[v1p])

				rs := nv.Text(v1)
				return rs
			case "value":
				if len(paramsA) < v1p+1 {
					return fmt.Errorf("????????????")
				}

				v1 := tk.ToInt(paramsA[v1p])

				rs := nv.Value(v1)
				return rs
			case "maxValue":
				rs := nv.MaxValue()
				return rs
			case "done":
				return nv.Done()
			}
		}

		rvr := tk.ReflectCallMethod(objT, methodNameT, paramsA[2:]...)

		return rvr

	case "new":
		if len(paramsA) < 1 {
			return fmt.Errorf("????????????")
		}

		vs1 := tk.ToStr(paramsA[0])

		p := objA.(*quixie.QuixieVM)

		switch vs1 {
		case "window", "webView2":
			return newWindowWebView2(p, paramsA[1:])
		}

		return fmt.Errorf("???????????????????????????%v", vs1)

	case "close":
		if len(paramsA) < 1 {
			return fmt.Errorf("????????????")
		}

		switch nv := paramsA[0].(type) {
		case zenity.ProgressDialog:
			nv.Close()
		}

		return ""

	case "showInfo":
		if len(paramsA) < 2 {
			return fmt.Errorf("????????????")
		}
		return showInfoGUI(tk.ToStr(paramsA[0]), tk.ToStr(paramsA[1]), paramsA[2:]...)

	case "showError":
		if len(paramsA) < 2 {
			return fmt.Errorf("????????????")
		}
		return showErrorGUI(tk.ToStr(paramsA[0]), tk.ToStr(paramsA[1]), paramsA[2:]...)

	case "getConfirm":
		if len(paramsA) < 2 {
			return fmt.Errorf("????????????")
		}
		return getConfirmGUI(tk.ToStr(paramsA[0]), tk.ToStr(paramsA[1]), paramsA[2:]...)
	case "getActiveDisplayCount":
		return screenshot.NumActiveDisplays()
	case "getScreenResolution":
		var paraArgsT []string = []string{}

		for i := 0; i < len(paramsA); i++ {
			paraArgsT = append(paraArgsT, tk.ToStr(paramsA[i]))
		}

		pT := objA.(*quixie.QuixieVM)

		formatT := pT.GetSwitchVarValue(pT.Running, paraArgsT, "-format=", "")

		idxStrT := pT.GetSwitchVarValue(pT.Running, paraArgsT, "-index=", "0")

		idxT := tk.StrToInt(idxStrT, 0)

		rectT := screenshot.GetDisplayBounds(idxT)

		if formatT == "" {
			return []interface{}{rectT.Max.X, rectT.Max.Y}
		} else if formatT == "raw" || formatT == "rect" {
			return rectT
		} else if formatT == "json" {
			return tk.ToJSONX(rectT, "-sort")
		}

		return []interface{}{rectT.Max.X, rectT.Max.Y}
	case "showProcess":
		var paraArgsT []string = []string{}

		for i := 0; i < len(paramsA); i++ {
			paraArgsT = append(paraArgsT, tk.ToStr(paramsA[i]))
		}

		optionsT := []zenity.Option{}

		titleT := tk.GetSwitch(paraArgsT, "-title=", "")

		if titleT != "" {
			optionsT = append(optionsT, zenity.Title(titleT))
		}

		okButtonT := tk.GetSwitch(paraArgsT, "-ok=", "")

		if titleT != "" {
			optionsT = append(optionsT, zenity.OKLabel(okButtonT))
		}

		cancelButtonT := tk.GetSwitch(paraArgsT, "-cancel=", "")

		if titleT != "" {
			optionsT = append(optionsT, zenity.CancelLabel(cancelButtonT))
		}

		if tk.IfSwitchExistsWhole(paraArgsT, "-noCancel") {
			optionsT = append(optionsT, zenity.NoCancel())
		}

		if tk.IfSwitchExistsWhole(paraArgsT, "-modal") {
			optionsT = append(optionsT, zenity.Modal())
		}

		if tk.IfSwitchExistsWhole(paraArgsT, "-pulsate") {
			optionsT = append(optionsT, zenity.Pulsate())
		}

		maxT := tk.GetSwitch(paraArgsT, "-max=", "")

		if maxT != "" {
			optionsT = append(optionsT, zenity.MaxValue(tk.ToInt(maxT, 100)))
		}

		dlg, errT := zenity.Progress(optionsT...)
		if errT != nil {
			return fmt.Errorf("????????????????????????failed to create progress dialog??????%v", errT)
		}

		return dlg

	case "newWindowSciter":
		if len(paramsA) < 3 {
			return fmt.Errorf("????????????")
		}

		var paraArgsT []string = []string{}

		for i := 3; i < len(paramsA); i++ {
			paraArgsT = append(paraArgsT, tk.ToStr(paramsA[i]))
		}

		fromFileT := tk.IfSwitchExistsWhole(paraArgsT, "-fromFile")

		titleT := tk.ToStr(paramsA[0])

		rectStrT := tk.ToStr(paramsA[1]) //tk.GetSwitchI(paramsA, "-rect=", "")

		var rectT *sciter.Rect

		if rectStrT == "" {
			rectT = sciter.DefaultRect
		} else {
			objT, errT := tk.FromJSON(rectStrT)

			if errT != nil {
				return fmt.Errorf("???????????????????????????????????????%v", errT)
			}

			var aryT []int

			switch nv := objT.(type) {
			case []int:
				aryT = nv
			case []float64:
				if len(nv) < 4 {
					return fmt.Errorf("???????????????????????????????????????%v", "??????????????????")
				}
				aryT = []int{tk.ToInt(nv[0]), tk.ToInt(nv[1]), tk.ToInt(nv[2]), tk.ToInt(nv[3])}
			case []interface{}:
				if len(nv) < 4 {
					return fmt.Errorf("???????????????????????????????????????%v", "??????????????????")
				}
				aryT = []int{tk.ToInt(nv[0]), tk.ToInt(nv[1]), tk.ToInt(nv[2]), tk.ToInt(nv[3])}
			}

			rectT = &sciter.Rect{Left: int32(aryT[0]), Top: int32(aryT[1]), Right: int32(aryT[0] + aryT[2]), Bottom: int32(aryT[1] + aryT[3])}

		}

		w, errT := window.New(sciter.DefaultWindowCreateFlag, rectT)

		if errT != nil {
			return fmt.Errorf("?????????????????????%v", errT)
		}

		w.SetOption(sciter.SCITER_SET_SCRIPT_RUNTIME_FEATURES, sciter.ALLOW_EVAL|sciter.ALLOW_SYSINFO|sciter.ALLOW_FILE_IO|sciter.ALLOW_SOCKET_IO)

		w.SetTitle(titleT)

		htmlT := tk.ToStr(paramsA[2])

		baseUrlT := tk.GetSwitch(paraArgsT, "-baseUrl=", "")

		// tk.Pln(fromFileT, htmlT, baseUrlT, tk.PathToURI("."))

		if fromFileT {
			htmlNewT, errT := filepath.Abs(htmlT)
			if errT == nil {
				htmlT = htmlNewT
			}

			errT = w.LoadFile(htmlT)

			if tk.IsErrX(errT) {
				return fmt.Errorf("????????????%v????????????????????????%v", htmlT, errT)
			}
		} else {
			htmlT := tk.ToStr(paramsA[2])

			if baseUrlT == "." {
				baseUrlT = tk.PathToURI(".") + "/basic.html"
			}

			w.LoadHtml(htmlT, baseUrlT)
		}

		var handlerT tk.TXDelegate

		handlerT = func(actionA string, objA interface{}, dataA interface{}, paramsA ...interface{}) interface{} {
			switch actionA {
			case "show":
				w.Show()
				w.Run()
				return nil
			case "setDelegate":
				var deleT tk.QuickDelegate = paramsA[0].(tk.QuickDelegate)

				w.DefineFunction("delegateDo", func(args ...*sciter.Value) *sciter.Value {
					// args???SciterJS??????????????????????????????????????????
					// ?????????????????????????????????????????????????????????
					strT := args[0].String()

					rsT := deleT(strT)

					// ??????????????????????????????????????????????????????
					return sciter.NewValue(rsT)
				})

				return nil
			case "call":
				len1T := len(paramsA)
				if len1T < 1 {
					return fmt.Errorf("????????????")
				}

				if len1T > 1 {
					aryT := make([]*sciter.Value, 0, 10)

					for i := 1; i < len1T; i++ {
						aryT = append(aryT, sciter.NewValue(paramsA[i]))
					}

					rsT, errT := w.Call(tk.ToStr(paramsA[0]), aryT...)

					if errT != nil {
						return fmt.Errorf("??????????????????????????????%v", errT)
					}

					return rsT.String()
				}

				rsT, errT := w.Call(tk.ToStr(paramsA[0]))

				if errT != nil {
					return fmt.Errorf("??????????????????????????????%v", errT)
				}

				return rsT.String()
			default:
				return fmt.Errorf("???????????????%v", actionA)
			}

			return nil
		}

		// w.Show()
		// w.Run()

		return handlerT

	case "newWindow":
		return newWindowWebView2(objA, paramsA)

	default:
		return fmt.Errorf("????????????")
	}

	return ""
}

func initGUI() error {
	applicationPathT := tk.GetApplicationPath()

	osT := tk.GetOSName()

	if tk.Contains(osT, "inux") {
	} else if tk.Contains(osT, "arwin") {
	} else {
		_, errT := exec.LookPath("sciter.dll")

		// tk.Pln("LookPath", errT)

		if errors.Is(errT, exec.ErrDot) {
			errT = nil
		}

		if errT != nil {
			if tk.IfFileExists("sciter.dll") || tk.IfFileExists(filepath.Join(applicationPathT, "sciter.dll")) {

			} else {
				tk.Pl("?????????WEB????????????????????????")
				rs := tk.DownloadFile("http://xie.topget.org/pub/sciter.dll", applicationPathT, "sciter.dll")

				if tk.IsErrorString(rs) {
					return fmt.Errorf("???????????????????????????????????????")
				}
			}
		}
	}

	// dialog.Do_init()
	// window.Do_init()

	return nil
}

func showInfoGUI(titleA string, formatA string, messageA ...interface{}) interface{} {
	rs, errT := dlgs.Info(titleA, fmt.Sprintf(formatA, messageA...))

	if errT != nil {
		return errT
	}

	return rs
}

func getConfirmGUI(titleA string, formatA string, messageA ...interface{}) interface{} {
	flagT, errT := dlgs.Question(titleA, fmt.Sprintf(formatA, messageA...), true)
	if errT != nil {
		return errT
	}

	return flagT
}

func showErrorGUI(titleA string, formatA string, messageA ...interface{}) interface{} {
	rs, errT := dlgs.Error(titleA, fmt.Sprintf(formatA, messageA...))
	if errT != nil {
		return errT
	}

	return rs
}
