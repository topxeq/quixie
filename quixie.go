package quixie

import (
	"encoding/json"
	"fmt"
	"image"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/topxeq/tk"
)

var VersionG string = "0.1.1"

func Test() {
	tk.Pl("test")
}

var InstrCodeSet map[int]string = map[int]string{}

var InstrNameSet map[string]int = map[string]int{

	// internal & debug related
	"invalidInstr": 12, // invalid instruction, use internally to indicate invalid instr(s) while parsing commands

	"version": 100, // get current Quixie version, return a string type value, if the result parameter not designated, it will be put to the global variable $tmp(and it's same for other instructions has result and not variable parameters)

	"pass": 101, // do nothing, useful for placeholder

	"debugInfo": 103, // get the debug info

	"test":             121, // for test purpose, check if 2 values are equal
	"testByStartsWith": 122, // for test purpose, check if first string starts with the 2nd
	"testByReg":        123, // for test purpose, check if first string matches the regex pattern defined by the 2nd string

	"compile": 153, // compile a piece of code

	"goto": 180, // jump to the instruction line (often indicated by labels)
	"jmp":  180,

	"exit": 199, // terminate the program, can with a return value(same as assign the global value $outG)

	// var related
	"global": 201, // define a global variable

	"var": 203, // define a local variable

	// push/peek/pop stack related

	"push": 220, // push any value to stack

	"peek": 222, // peek the value on the top of the stack

	"pop": 224, // pop the value on the top of the stack

	"getStackSize": 230,

	"clearStack": 240,

	// assign related
	"assign": 401, // assignment, from local variable to global, assign value to local if not found
	"=":      401,

	// operator related

	"inc": 801, // ++
	"++":  801,

	"dec": 810, // --
	"--":  810,

	"add": 901, // add 2 values
	"+":   901,

	// func related

	"call": 1010, // call a normal function

	"ret": 1020, // return from a normal function

	"sealCall": 1055, //

	"fastCall": 1070, // fast call function, no function stack used

	"fastRet": 1071, // return from fast function, used with fastCall

	// array/slice related 数组/切片相关

	"getArrayItem": 1123,
	"[]":           1123,

	// print related
	"pln": 10410, // same as println function in other languages

	"plv": 10430,

	// system related
	"systemCmd": 20601,
}

// type UndefinedStruct struct {
// 	int
// }

// func (o UndefinedStruct) String() string {
// 	return "undefined"
// }

// var Undefined UndefinedStruct = UndefinedStruct{0}

// type VarRef struct {
// 	Type  int // 1, 3: any value, 7: label, 9xx: predefined variables,
// 	Name  string
// 	Value interface{}
// }

type VarRef struct {
	Ref   int // -99 - invalid, -15 - ref, -12 - unref, -11 - seq, -10 - quickEval, -9 - eval, -8 - pop, -7 - peek, -6 - push, -5 - tmp, -4 - pln, -3 - value only, -2 - drop, -1 - debug, 3 normal vars
	Value interface{}
}

type Instr struct {
	Code     int
	Cmd      string
	ParamLen int
	Params   []VarRef
	Line     string
}

type GlobalContext struct {
	SyncMap   tk.SyncMap
	SyncQueue tk.SyncQueue
	SyncStack tk.SyncStack

	SyncSeq tk.Seq

	Vars map[string]interface{}

	VerboseLevel int
}

type FuncContext struct {
	Vars map[string]interface{}

	Tmp interface{}

	// ReturnPointer int

	DeferStack *tk.SimpleStack

	// Layer int

	// Parent *FuncContext
}

func NewFuncContext() *FuncContext {
	rs := &FuncContext{}

	rs.Vars = make(map[string]interface{})

	rs.DeferStack = tk.NewSimpleStack(10, tk.Undefined)

	return rs
}

type CompiledCode struct {
	Labels map[string]int

	Source []string

	CodeList []string

	InstrList []Instr

	CodeSourceMap map[int]int
}

func NewCompiledCode() *CompiledCode {
	ccT := &CompiledCode{}

	ccT.Labels = make(map[string]int, 0)

	ccT.Source = make([]string, 0)

	ccT.CodeList = make([]string, 0)

	ccT.InstrList = make([]Instr, 0)

	ccT.CodeSourceMap = make(map[int]int, 0)

	return ccT
}

func ParseLine(commandA string) ([]string, string, error) {
	var args []string
	var lineT string

	firstT := true

	// state: 1 - start, quotes - 2, arg - 3
	state := 1
	current := ""
	quote := "`"
	// escapeNext := false

	command := []rune(commandA)

	for i := 0; i < len(command); i++ {
		c := command[i]

		// if escapeNext {
		// 	current += string(c)
		// 	escapeNext = false
		// 	continue
		// }

		// if c == '\\' {
		// 	current += string(c)
		// 	escapeNext = true
		// 	continue
		// }

		if state == 2 {
			if string(c) != quote {
				current += string(c)
			} else {
				current += string(c) // add it

				args = append(args, current)
				if firstT {
					firstT = false
					lineT = string(command[i:])
				}
				current = ""
				state = 1
			}
			continue
		}

		// tk.Pln(string(c), c, c == '`', '`')
		if c == '"' || c == '\'' || c == '`' {
			state = 2
			quote = string(c)

			current += string(c) // add it

			continue
		}

		if state == 3 {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				if firstT {
					firstT = false
					lineT = string(command[i:])
				}
				current = ""
				state = 1
			} else {
				current += string(c)
			}
			// Pl("state: %v, current: %v, args: %v", state, current, args)
			continue
		}

		if c != ' ' && c != '\t' {
			state = 3
			current += string(c)
		}
	}

	if state == 2 {
		return []string{}, lineT, fmt.Errorf("unclosed quotes: %v", string(command))
	}

	if current != "" {
		args = append(args, current)
		if firstT {
			firstT = false
			lineT = ""
		}
	}

	return args, lineT, nil
}

func ParseVar(strA string, optsA ...interface{}) VarRef {
	s1T := strings.TrimSpace(strA)

	if strings.HasPrefix(s1T, "`") && strings.HasSuffix(s1T, "`") {
		s1T = s1T[1 : len(s1T)-1]

		return VarRef{-3, s1T} // value(string)
	} else if strings.HasPrefix(s1T, `"`) && strings.HasSuffix(s1T, `"`) {
		tmps, errT := strconv.Unquote(s1T)

		if errT != nil {
			return VarRef{-3, s1T}
		}

		return VarRef{-3, tmps} // value(string)
	} else {
		if strings.HasPrefix(s1T, "$") {
			var vv interface{} = nil

			if strings.HasSuffix(s1T, "...") {
				vv = "..."

				s1T = s1T[:len(s1T)-3]
			}

			if s1T == "$drop" {
				return VarRef{-2, nil}
			} else if s1T == "$debug" {
				return VarRef{-1, nil}
			} else if s1T == "$pln" {
				return VarRef{-4, nil}
			} else if s1T == "$pop" {
				return VarRef{-8, vv}
			} else if s1T == "$peek" {
				return VarRef{-7, vv}
			} else if s1T == "$push" {
				return VarRef{-6, nil}
			} else if s1T == "$tmp" {
				return VarRef{-5, vv}
			} else if s1T == "$seq" {
				return VarRef{-11, nil}
			} else {
				return VarRef{3, s1T[1:]}
			}
		} else if strings.HasPrefix(s1T, "#") { // values
			if len(s1T) < 2 {
				return VarRef{-3, s1T}
			}

			// remainsT := s1T[2:]

			typeT := s1T[1]

			if typeT == 'i' { // int
				c1T, errT := tk.StrToIntQuick(s1T[2:])

				if errT != nil {
					return VarRef{-3, s1T}
				}

				return VarRef{-3, c1T}
			} else if typeT == 'f' { // float
				c1T, errT := tk.StrToFloat64E(s1T[2:])

				if errT != nil {
					return VarRef{-3, s1T}
				}

				return VarRef{-3, c1T}
			} else if typeT == 'b' { // bool
				return VarRef{-3, tk.ToBool(s1T[2:])}
			} else if typeT == 'y' { // byte
				return VarRef{-3, tk.ToByte(s1T[2:])}
			} else if typeT == 'B' { // single rune (same as in Golang, like 'a'), only first character in string is used
				runesT := []rune(s1T[2:])

				if len(runesT) < 1 {
					return VarRef{-3, s1T[2:]}
				}

				return VarRef{-3, runesT[0]}
			} else if typeT == 'r' { // rune
				return VarRef{-3, tk.ToRune(s1T[2:])}
			} else if typeT == 's' { // string
				s1DT := s1T[2:]

				if strings.HasPrefix(s1DT, "`") && strings.HasSuffix(s1DT, "`") {
					s1DT = s1DT[1 : len(s1DT)-1]
				}

				return VarRef{-3, tk.ToStr(s1DT)}
			} else if typeT == 'e' { // error
				s1DT := s1T[2:]

				if strings.HasPrefix(s1DT, "`") && strings.HasSuffix(s1DT, "`") {
					s1DT = s1DT[1 : len(s1DT)-1]
				}

				return VarRef{-3, fmt.Errorf("%v", s1DT)}
			} else if typeT == 't' { // time
				s1DT := s1T[2:]

				if strings.HasPrefix(s1DT, "`") && strings.HasSuffix(s1DT, "`") {
					s1DT = s1DT[1 : len(s1DT)-1]
				}

				tmps := strings.TrimSpace(s1DT)

				if tmps == "" || tmps == "now" {
					return VarRef{-3, time.Now()}
				}

				rsT := tk.ToTime(tmps)

				if tk.IsError(rsT) {
					return VarRef{-3, s1T}
				}

				return VarRef{-3, rsT}
			} else if typeT == 'L' { // list/array
				var listT []interface{}

				s1DT := s1T[2:] // tk.UrlDecode(s1T[2:])

				if strings.HasPrefix(s1DT, "`") && strings.HasSuffix(s1DT, "`") {
					s1DT = s1DT[1 : len(s1DT)-1]
				}

				// tk.Plv(s1T[2:])
				// tk.Plv(s1DT)

				errT := json.Unmarshal([]byte(s1DT), &listT)
				// tk.Plv(errT)
				if errT != nil {
					return VarRef{-3, s1T}
				}

				// tk.Plv(listT)
				return VarRef{-3, listT}
			} else if typeT == 'Y' { // byteList
				var listT []byte

				s1DT := s1T[2:] // tk.UrlDecode(s1T[2:])

				if strings.HasPrefix(s1DT, "`") && strings.HasSuffix(s1DT, "`") {
					s1DT = s1DT[1 : len(s1DT)-1]
				}

				// tk.Plv(s1T[2:])
				// tk.Plv(s1DT)

				errT := json.Unmarshal([]byte(s1DT), &listT)
				// tk.Plv(errT)
				if errT != nil {
					return VarRef{-3, s1T}
				}

				// tk.Plv(listT)
				return VarRef{-3, listT}
			} else if typeT == 'R' { // runeList
				var listT []rune

				s1DT := s1T[2:] // tk.UrlDecode(s1T[2:])

				if strings.HasPrefix(s1DT, "`") && strings.HasSuffix(s1DT, "`") {
					s1DT = s1DT[1 : len(s1DT)-1]
				}

				// tk.Plv(s1T[2:])
				// tk.Plv(s1DT)

				errT := json.Unmarshal([]byte(s1DT), &listT)
				// tk.Plv(errT)
				if errT != nil {
					return VarRef{-3, s1T}
				}

				// tk.Plv(listT)
				return VarRef{-3, listT}
			} else if typeT == 'S' { // strList/stringList
				var listT []string

				s1DT := s1T[2:] // tk.UrlDecode(s1T[2:])

				if strings.HasPrefix(s1DT, "`") && strings.HasSuffix(s1DT, "`") {
					s1DT = s1DT[1 : len(s1DT)-1]
				}

				// tk.Plv(s1T[2:])
				// tk.Plv(s1DT)

				errT := json.Unmarshal([]byte(s1DT), &listT)
				// tk.Plv(errT)
				if errT != nil {
					return VarRef{-3, s1T}
				}

				// tk.Plv(listT)
				return VarRef{-3, listT}
			} else if typeT == 'M' { // map
				var mapT map[string]interface{}

				s1DT := s1T[2:] // tk.UrlDecode(s1T[2:])

				if strings.HasPrefix(s1DT, "`") && strings.HasSuffix(s1DT, "`") {
					s1DT = s1DT[1 : len(s1DT)-1]
				}

				// tk.Plv(s1T[2:])
				// tk.Plv(s1DT)

				errT := json.Unmarshal([]byte(s1DT), &mapT)
				// tk.Plv(errT)
				if errT != nil {
					return VarRef{-3, s1T}
				}

				// tk.Plv(listT)
				return VarRef{-3, mapT}
			}

			return VarRef{-3, s1T}
		}
	}

	return VarRef{-3, s1T}
}

func Compile(codeA string) interface{} {
	p := NewCompiledCode()

	originCodeLenT := 0

	sourceT := tk.SplitLines(codeA)

	p.Source = append(p.Source, sourceT...)

	pointerT := originCodeLenT

	for i := 0; i < len(sourceT); i++ {
		v := strings.TrimSpace(sourceT[i])

		if tk.StartsWith(v, "//") || tk.StartsWith(v, "#") {
			continue
		}

		if tk.StartsWith(v, ":") {
			labelT := strings.TrimSpace(v[1:])

			_, ok := p.Labels[labelT]

			if !ok {
				p.Labels[labelT] = pointerT
			} else {
				return fmt.Errorf("编译错误(行 %v %v): 重复的标号", i+1, tk.LimitString(p.Source[i], 50))
			}

			continue
		}

		iFirstT := i
		if tk.Contains(v, "`") {
			if strings.Count(v, "`")%2 != 0 {
				foundT := false
				var j int
				for j = i + 1; j < len(sourceT); j++ {
					if tk.Contains(sourceT[j], "`") {
						v = tk.JoinLines(sourceT[i : j+1])
						foundT = true
						break
					}
				}

				if !foundT {
					return fmt.Errorf("代码解析错误: ` 未成对(%v)", i)
				}

				i = j
			}
		}

		v = strings.TrimSpace(v)

		if v == "" {
			continue
		}

		p.CodeList = append(p.CodeList, v)
		p.CodeSourceMap[pointerT] = originCodeLenT + iFirstT
		pointerT++
	}

	for i := originCodeLenT; i < len(p.CodeList); i++ {
		// listT := strings.SplitN(v, " ", 3)
		v := p.CodeList[i]

		listT, lineT, errT := ParseLine(v)
		if errT != nil {
			return fmt.Errorf("参数解析失败：%v", errT)
		}

		lenT := len(listT)

		instrNameT := strings.TrimSpace(listT[0])

		codeT, ok := InstrNameSet[instrNameT]

		if !ok {
			instrT := Instr{Code: codeT, Cmd: InstrCodeSet[codeT], ParamLen: 1, Params: []VarRef{VarRef{Ref: -3, Value: v}}, Line: lineT} //&([]VarRef{})}
			p.InstrList = append(p.InstrList, instrT)

			return fmt.Errorf("编译错误(行 %v/%v %v): 未知指令", i, p.CodeSourceMap[i]+1, tk.LimitString(p.Source[p.CodeSourceMap[i]], 50))
		}

		instrT := Instr{Code: codeT, Cmd: InstrCodeSet[codeT], Params: make([]VarRef, 0, lenT-1), Line: lineT} //&([]VarRef{})}

		list3T := []VarRef{}

		for j, jv := range listT {
			if j == 0 {
				continue
			}

			list3T = append(list3T, ParseVar(jv, i))
		}

		instrT.Params = append(instrT.Params, list3T...)
		instrT.ParamLen = lenT - 1

		p.InstrList = append(p.InstrList, instrT)
	}

	// tk.Plv(p.SourceM)
	// tk.Plv(p.CodeListM)
	// tk.Plv(p.CodeSourceMapM)

	return p
}

type RunningContext struct {
	Labels map[string]int

	Source []string

	CodeList []string

	InstrList []Instr

	CodeSourceMap map[int]int

	CodePointer int

	PointerStack *tk.SimpleStack

	FuncStack *tk.SimpleStack

	// Parent interface{}
}

func (p *RunningContext) Initialize() {
	p.Labels = make(map[string]int)
	p.Source = make([]string, 0)
	p.CodeList = make([]string, 0)
	p.InstrList = make([]Instr, 0)
	p.CodeSourceMap = make(map[int]int)
	p.CodePointer = 0

	p.PointerStack = tk.NewSimpleStack(10, tk.Undefined)

	p.FuncStack = tk.NewSimpleStack(10, tk.Undefined)
}

func (p *RunningContext) LoadCompiled(compiledA *CompiledCode) error {
	// tk.Plv(compiledA)

	// load labels
	originalLenT := len(p.CodeList)

	for k, _ := range compiledA.Labels {
		_, ok := p.Labels[k]

		if ok {
			return fmt.Errorf("duplicate label: %v", k)
		}
	}

	for k, v := range compiledA.Labels {
		p.Labels[k] = originalLenT + v
	}

	// load codeList, instrList
	p.Source = append(p.Source, compiledA.Source...)

	p.CodeList = append(p.CodeList, compiledA.CodeList...)
	// for _, v := range compiledA.CodeList {
	// 	p.CodeList = append(p.CodeList, v)
	// }

	p.InstrList = append(p.InstrList, compiledA.InstrList...)
	// for _, v := range compiledA.InstrList {
	// 	p.InstrList = append(p.InstrList, v)
	// }

	for k, v := range compiledA.CodeSourceMap {
		p.CodeSourceMap[originalLenT+k] = originalLenT + v
	}

	return nil
}

func (p *RunningContext) LoadCode(codeA string) error {
	rs := Compile(codeA)

	if e1, ok := rs.(error); ok {
		return e1
	}

	e2 := p.LoadCompiled(rs.(*CompiledCode))

	if e2 != nil {
		return e2
	}

	return nil
}

func (p *RunningContext) Load(inputA ...interface{}) error {

	var inputT interface{} = nil

	if len(inputA) > 0 {
		inputT = inputA[0]
	}

	if inputT == nil {
		return nil
	}

	c1, ok := inputT.(*CompiledCode)

	if ok {
		errT := p.LoadCompiled(c1)

		if errT != nil {
			return errT
		}

		return nil
	}

	s1, ok := inputT.(string)
	if ok {
		e2 := p.LoadCode(s1)

		if e2 != nil {
			return e2
		}

		return nil
	}

	return fmt.Errorf("invalid parameter: %#v", inputT)
}

// return <0 as error
func (p *RunningContext) GetLabelIndex(inputA interface{}) int {
	c, ok := inputA.(int)

	if ok {
		return c
	}

	s2 := tk.ToStr(inputA)

	if len(s2) > 1 && s2[0:1] == ":" {
		s2a := s2[1:]
		if strings.HasPrefix(s2a, "+") {
			return p.CodePointer + tk.ToInt(s2a[1:])
		} else if strings.HasPrefix(s2, "-") {
			return p.CodePointer - tk.ToInt(s2a[1:])
		} else {
			labelPointerT, ok := p.Labels[s2a]

			if ok {
				return labelPointerT
			}
		}
	}

	return -1
}

func (p *RunningContext) GetFuncContext(layerT int) *FuncContext {
	if layerT < 0 {
		return p.FuncStack.Peek().(*FuncContext)
	}

	return p.FuncStack.PeekLayer(layerT).(*FuncContext)
}

func (p *RunningContext) GetCurrentFuncContext() *FuncContext {
	return p.FuncStack.Peek().(*FuncContext)
}

func (p *FuncContext) RunDefer(vmA *QuixieVM, rcA *RunningContext) error {
	for {
		instrT := p.DeferStack.Pop()

		// tk.Pl("\nDeferStack.Pop: %#v\n", instrT)

		if instrT == nil || tk.IsUndefined(instrT) {
			break
		}

		nv, ok := instrT.(*Instr)

		if !ok {
			return fmt.Errorf("invalid instruction: %#v", instrT)
		}

		if GlobalsG.VerboseLevel > 1 {
			tk.Pl("defer run: %v", nv)
		}

		rs := RunInstr(vmA, rcA, nv)

		if tk.IsErrX(rs) {
			return fmt.Errorf("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStrX(rs))
		}
	}

	return nil
}

func (p *QuixieVM) RunDefer(runA *RunningContext) error {
	if runA == nil {
		runA = p.Running
	}

	var currentFuncT *FuncContext

	lenT := runA.FuncStack.Size()

	if lenT < 1 {
		currentFuncT = p.RootFunc
	}

	rs := currentFuncT.RunDefer(p, runA)

	return rs
}

func (p *QuixieVM) RunDeferUpToRoot(runA *RunningContext) error {
	if runA == nil {
		runA = p.Running
	}

	lenT := runA.FuncStack.Size()

	if lenT > 0 {
		rs := runA.RunDeferUpToRoot(p)

		if tk.IsError(rs) {
			return rs
		}
	}

	rs := p.RootFunc.RunDefer(p, runA)

	return rs
}

func (p *RunningContext) RunDeferUpToRoot(vmA *QuixieVM) error {
	// if p.Parent == nil {
	// 	return fmt.Errorf("no parent VM: %v", p.Parent)
	// }

	lenT := p.FuncStack.Size()

	if lenT < 1 {
		return nil
	}

	for i := lenT - 1; i >= 0; i-- {
		contextT := p.FuncStack.PeekLayer(i).(*FuncContext)

		rs := contextT.RunDefer(vmA, p)

		if tk.IsError(rs) {
			return rs
		}
	}

	return nil

}

func (p *RunningContext) RunDefer(vmA *QuixieVM) error {
	// if p.Parent == nil {
	// 	return fmt.Errorf("no parent VM: %v", p.Parent)
	// }

	lenT := p.FuncStack.Size()

	if lenT < 1 {
		return nil
	}

	contextT := p.FuncStack.Peek().(*FuncContext)

	rs := contextT.RunDefer(vmA, p)

	if tk.IsError(rs) {
		return rs
	}

	return nil

	// for {
	// 	instrT := p.GetCurrentFuncContext().DeferStack.Pop()

	// 	// tk.Pl("\nDeferStack.Pop: %#v\n", instrT)

	// 	if instrT == nil || tk.IsUndefined(instrT) {
	// 		break
	// 	}

	// 	nv, ok := instrT.(*Instr)

	// 	if !ok {
	// 		return fmt.Errorf("invalid instruction: %#v", instrT)
	// 	}

	// 	if GlobalsG.VerboseLevel > 1 {
	// 		tk.Pl("defer run: %v", nv)
	// 	}

	// 	rs := RunInstr(vmA, p, nv)

	// 	if tk.IsErrX(rs) {
	// 		return fmt.Errorf("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStrX(rs))
	// 	}
	// }

	// return nil
}

// func (p *RunningContext) RunDeferUpToRoot() interface{} {
// 	if p.Parent == nil {
// 		return fmt.Errorf("no parent VM: %v", p.Parent)
// 	}

// 	currentContextIndexT := p.FuncStack.Size()

// 	var currentFuncT *FuncContext

// 	for {
// 		// tk.Pl("currentContextT: %#v", currentContextT)

// 		if currentContextIndexT > 0 {
// 			currentFuncT = p.FuncStack.PeekLayer(currentContextIndexT).(*FuncContext)
// 		} else {
// 			currentFuncT = p.Parent.(*QuixieVM).RootFunc
// 		}

// 		instrT := currentFuncT.DeferStack.Pop()

// 		// tk.Pl("\nDeferStack.Pop: %#v\n", instrT)

// 		if instrT == nil || tk.IsUndefined(instrT) {
// 			if currentContextIndexT < 1 {
// 				break
// 			}

// 			currentContextIndexT--

// 			continue
// 		}

// 		nv, ok := instrT.(*Instr)

// 		if !ok {
// 			return fmt.Errorf("invalid instruction: %#v", instrT)
// 		}

// 		if GlobalsG.VerboseLevel > 1 {
// 			tk.Pl("defer run: %v", nv)
// 		}

// 		rs := RunInstr(p.Parent.(*QuixieVM), p, nv)

// 		if tk.IsErrX(rs) {
// 			return fmt.Errorf("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStrX(rs))
// 		}
// 	}

// 	return nil
// }

func NewRunningContext(inputA ...interface{}) interface{} {
	var inputT interface{} = nil

	if len(inputA) > 0 {
		inputT = inputA[0]
	}

	rc := &RunningContext{}

	rc.Initialize()

	if inputT == nil {
		return rc
	}

	c1, ok := inputT.(*CompiledCode)

	if ok {
		errT := rc.LoadCompiled(c1)

		if errT != nil {
			return errT
		}

		return rc
	}

	s1, ok := inputT.(string)
	if ok {
		e2 := rc.LoadCode(s1)

		if e2 != nil {
			return e2
		}

		return rc
	}

	return fmt.Errorf("invalid parameter: %#v", inputT)
}

type QuixieVM struct {
	Regs  []interface{}
	Stack *tk.SimpleStack

	// Vars map[string]interface{}

	RootFunc *FuncContext

	Running *RunningContext
}

func NewVM(inputA ...interface{}) interface{} {
	var inputT interface{} = nil

	if len(inputA) > 0 {
		inputT = inputA[0]
	}

	rs := &QuixieVM{}

	rs.Regs = make([]interface{}, 20)
	rs.Stack = tk.NewSimpleStack(10, tk.Undefined)

	rs.RootFunc = NewFuncContext()
	// rs.Vars = make(map[string]interface{}, 0)

	var runningT interface{}

	if inputT != nil {
		runningT = NewRunningContext(inputT)
	} else {
		runningT = NewRunningContext()
	}

	if tk.IsError(runningT) {
		return runningT
	}

	rs.Running = runningT.(*RunningContext)

	// rs.Running.Parent = rs

	// rs.FuncStack = tk.NewSimpleStack(10, tk.Undefined)

	// rs.Running.FuncStack.Push(NewFuncContext())

	// set global variables
	rs.SetVar(rs.Running, "backQuote", "`")
	rs.SetVar(rs.Running, "undefined", tk.Undefined)
	rs.SetVar(rs.Running, "nil", nil)
	rs.SetVar(rs.Running, "newLine", "\n")
	// rs.SetVar("tmp", "")

	return rs
}

func NewVMQuick(inputA ...interface{}) *QuixieVM {
	rs := NewVM(inputA...)

	if tk.IsError(rs) {
		return nil
	}

	return rs.(*QuixieVM)
}

func (p *QuixieVM) GetCodeLen(runA *RunningContext) int {
	if runA == nil {
		runA = p.Running
	}

	return len(runA.CodeList)
}

func (p *QuixieVM) GetSwitchVarValue(runA *RunningContext, argsA []string, switchStrA string, defaultA ...string) string {
	if runA == nil {
		runA = p.Running
	}

	vT := tk.GetSwitch(argsA, switchStrA, defaultA...)

	vr := ParseVar(vT)

	return tk.ToStr(p.GetVarValue(runA, vr))
}

func (p *QuixieVM) GetFuncContext(runA *RunningContext, layerT int) *FuncContext {
	if runA == nil {
		runA = p.Running
	}

	if runA.FuncStack.Size() < 1 {
		return p.RootFunc
	}

	if layerT < 0 {
		return runA.FuncStack.Peek().(*FuncContext)
	}

	if layerT == 0 {
		return p.RootFunc
	}

	if layerT-1 >= 0 && layerT-1 < runA.FuncStack.Size() {
		return runA.FuncStack.PeekLayer(layerT - 1).(*FuncContext)
	}

	return nil
}

func (p *QuixieVM) GetCurrentFuncContext(runA *RunningContext) *FuncContext {
	if runA == nil {
		runA = p.Running
	}

	if runA.FuncStack.Size() < 1 {
		return p.RootFunc
	}

	return runA.FuncStack.Peek().(*FuncContext)
}

func (p *RunningContext) PushFunc() {
	funcContextT := NewFuncContext()

	// funcContextT.ReturnPointer = p.CodePointer + 1
	// funcContextT.Layer = p.FuncStack.Size() + 1

	// &FuncContext{Vars: make(map[string]interface{}, 0), ReturnPointer: p.CodePointer + 1, Layer: p.FuncStack.Size(), DeferStack: tk.NewSimpleStack(10, tk.Undefined)} // , Parent: p.FuncStack.Peek().(*FuncContext)

	p.FuncStack.Push(funcContextT)
}

func (p *RunningContext) PopFunc() error {

	funcContextItemT := p.FuncStack.Pop()

	if tk.IsUndefined(funcContextItemT) {
		return fmt.Errorf("no function in func stack")
	}

	// funcContextT := funcContextItemT.(*FuncContext)

	return nil
}

func (p *QuixieVM) GetVarValue(runA *RunningContext, vA VarRef) interface{} {
	if runA == nil {
		runA = p.Running
	}

	idxT := vA.Ref

	if idxT == -2 {
		return tk.Undefined
	}

	if idxT == -3 {
		return vA.Value
	}

	if idxT == -5 {
		return p.GetCurrentFuncContext(runA).Tmp
	}

	if idxT == -11 {
		return GlobalsG.SyncSeq.Get()
	}

	if idxT == -8 {
		return p.Stack.Pop()
	}

	if idxT == -7 {
		return p.Stack.Peek()
	}

	if idxT == -1 { // $debug
		return tk.ToJSONX(p, "-indent", "-sort")
	}

	if idxT == -6 {
		return tk.Undefined
	}

	if idxT == 3 { // normal variables
		lenT := runA.FuncStack.Size()

		for idxT := lenT - 1; idxT >= 0; idxT-- {
			loopFunc := runA.FuncStack.PeekLayer(idxT).(*FuncContext)
			nv, ok := loopFunc.Vars[vA.Value.(string)]

			if ok {
				return nv
			}
		}

		nv, ok := p.RootFunc.Vars[vA.Value.(string)]

		if ok {
			return nv
		}

		return tk.Undefined

	}

	return tk.Undefined

}

func (p *QuixieVM) SetVar(runA *RunningContext, refA interface{}, setValueA interface{}) error {
	if runA == nil {
		runA = p.Running
	}

	// tk.Pln(refA, "->", setValueA)
	if refA == nil {
		return fmt.Errorf("nil parameter")
	}

	var refT VarRef

	c1, ok := refA.(int)

	if ok {
		refT = VarRef{Ref: c1, Value: nil}
	} else {
		s1, ok := refA.(string)

		if ok {
			refT = VarRef{Ref: 3, Value: s1}
		} else {
			r1, ok := refA.(VarRef)

			if ok {
				refT = r1
			} else {

				r2, ok := refA.(*VarRef)

				if ok {
					refT = *r2
				}
			}
		}
	}

	refIntT := refT.Ref

	currentFunc := p.GetCurrentFuncContext(runA)

	if refIntT == -2 { // $drop
		return nil
	}

	if refIntT == -4 { // $pln
		fmt.Println(setValueA)
		return nil
	}

	if refIntT == -5 { // $tmp
		currentFunc.Tmp = setValueA
		return nil
	}

	if refIntT == -6 { // $push
		p.Stack.Push(setValueA)
		return nil
	}

	if refIntT == -11 { // $seq
		GlobalsG.SyncSeq.Reset(tk.ToInt(setValueA, 0))
		return nil
	}

	if refIntT != 3 {
		return fmt.Errorf("unsupported var reference")
	}

	lenT := runA.FuncStack.Size()

	keyT := refT.Value.(string)

	for idxT := lenT - 1; idxT >= 0; idxT-- {
		loopFunc := runA.FuncStack.PeekLayer(idxT).(*FuncContext)
		_, ok := loopFunc.Vars[keyT]

		if ok {
			loopFunc.Vars[keyT] = setValueA

			return nil
		}
	}

	_, ok = p.RootFunc.Vars[keyT]

	if ok {
		p.RootFunc.Vars[keyT] = setValueA
		return nil
	}

	currentFunc.Vars[keyT] = setValueA

	return nil
}

func (p *QuixieVM) SetVarLocal(runA *RunningContext, refA interface{}, setValueA interface{}) error {
	if runA == nil {
		runA = p.Running
	}

	// tk.Pln(refA, "->", setValueA)
	if refA == nil {
		return fmt.Errorf("nil parameter")
	}

	var refT VarRef

	c1, ok := refA.(int)

	if ok {
		refT = VarRef{Ref: c1, Value: nil}
	} else {
		s1, ok := refA.(string)

		if ok {
			refT = VarRef{Ref: 3, Value: s1}
		} else {
			r1, ok := refA.(VarRef)

			if ok {
				refT = r1
			} else {

				r2, ok := refA.(*VarRef)

				if ok {
					refT = *r2
				}
			}
		}
	}

	refIntT := refT.Ref

	currentFunc := p.GetCurrentFuncContext(runA)

	if refIntT == -2 { // $drop
		return nil
	}

	if refIntT == -4 { // $pln
		fmt.Println(setValueA)
		return nil
	}

	if refIntT == -5 { // $tmp
		currentFunc.Tmp = setValueA
		return nil
	}

	if refIntT == -6 { // $push
		p.Stack.Push(setValueA)
		return nil
	}

	if refIntT == -11 { // $seq
		GlobalsG.SyncSeq.Reset(tk.ToInt(setValueA, 0))
		return nil
	}

	if refIntT != 3 {
		return fmt.Errorf("unsupported var reference")
	}

	keyT := refT.Value.(string)

	currentFunc.Vars[keyT] = setValueA

	return nil
}

func (p *QuixieVM) SetVarGlobal(refA interface{}, setValueA interface{}) error {
	// tk.Plv(refA)
	if refA == nil {
		return fmt.Errorf("nil parameter")
	}

	var refT VarRef

	c1, ok := refA.(int)

	if ok {
		refT = VarRef{Ref: c1, Value: nil}
	} else {
		s1, ok := refA.(string)

		if ok {
			refT = VarRef{Ref: 3, Value: s1}
		} else {
			r1, ok := refA.(VarRef)

			if ok {
				refT = r1
			} else {

				r2, ok := refA.(*VarRef)

				if ok {
					refT = *r2
				}
			}
		}
	}

	refIntT := refT.Ref

	currentFunc := p.GetFuncContext(nil, 0)

	if refIntT == -2 { // $drop
		return nil
	}

	if refIntT == -4 { // $pln
		fmt.Println(setValueA)
		return nil
	}

	if refIntT == -5 { // $tmp
		currentFunc.Tmp = setValueA
		return nil
	}

	if refIntT == -6 { // $push
		p.Stack.Push(setValueA)
		return nil
	}

	if refIntT == -11 { // $seq
		GlobalsG.SyncSeq.Reset(tk.ToInt(setValueA, 0))
		return nil
	}

	if refIntT != 3 {
		return fmt.Errorf("unsupported var reference")
	}

	keyT := refT.Value.(string)

	currentFunc.Vars[keyT] = setValueA

	return nil
}

// func (p *QuixieVM) SetVarQuick(keyA string, vA interface{}) error {

// 	lenT := p.FuncStack.Size()

// 	for idxT := lenT - 1; idxT >= 0; idxT-- {
// 		currentFunc := p.FuncStack.PeekLayer(idxT).(*FuncContext)

// 		_, ok := currentFunc.Vars[keyA]

// 		if ok {
// 			currentFunc.Vars[keyA] = vA

// 			return nil
// 		}
// 	}

// 	currentFunc := p.FuncStack.Peek().(*FuncContext)

// 	currentFunc.Vars[keyA] = vA

// 	return nil
// }

// func IsNoResult(inputA interface{}) bool {
// 	if inputA == nil {
// 		return false
// 	}

// 	nv, ok := inputA.(error)
// 	if !ok {
// 		return false
// 	}

// 	if nv.Error() == "no result" {
// 		return true
// 	}

// 	return false
// }

func (p *QuixieVM) Load(runA *RunningContext, codeA interface{}) error {
	if runA == nil {
		runA = p.Running
	}

	return runA.Load(codeA)
}

func (p *QuixieVM) LoadCompiled(runA *RunningContext, compiledA *CompiledCode) interface{} {
	if runA == nil {
		runA = p.Running
	}

	return runA.LoadCompiled(compiledA)
	// tk.Plv(compiledA)

	// // load labels
	// originalLenT := len(p.CodeList)

	// for k, _ := range compiledA.Labels {
	// 	_, ok := p.runningContext.Labels[k]

	// 	if ok {
	// 		return fmt.Errorf("duplicate label: %v", k)
	// 	}
	// }

	// for k, v := range compiledA.Labels {
	// 	p.runningContext.Labels[k] = originalLenT + v
	// }

	// // load codeList, instrList
	// p.Source = append(p.Source, compiledA.Source...)

	// p.CodeList = append(p.CodeList, compiledA.CodeList...)
	// // for _, v := range compiledA.CodeList {
	// // 	p.CodeList = append(p.CodeList, v)
	// // }

	// p.InstrList = append(p.InstrList, compiledA.InstrList...)
	// // for _, v := range compiledA.InstrList {
	// // 	p.InstrList = append(p.InstrList, v)
	// // }

	// for k, v := range compiledA.CodeSourceMap {
	// 	p.CodeSourceMap[originalLenT+k] = originalLenT + v
	// }

	// return nil
}

func (p *QuixieVM) Errf(runA *RunningContext, formatA string, argsA ...interface{}) error {
	if runA == nil {
		runA = p.Running
	}

	// tk.Pl("dbg: %v", tk.ToJSONX(p, "-sort"))
	// if p.VerbosePlusM {
	// 	tk.Pl(fmt.Sprintf("TXERROR:(Line %v: %v) ", p.CodeSourceMapM[p.CodePointerM]+1, tk.LimitString(p.SourceM[p.CodeSourceMapM[p.CodePointerM]], 50))+formatA, argsA...)
	// }

	return runA.Errf(formatA, argsA...)

	// return fmt.Errorf(fmt.Sprintf("(Line %v: %v) ", p.Running.CodeSourceMap[p.Running.CodePointer]+1, tk.LimitString(p.Running.Source[p.Running.CodeSourceMap[p.Running.CodePointer]], 50))+formatA, argsA...)
}

func (p *RunningContext) Errf(formatA string, argsA ...interface{}) error {
	return fmt.Errorf(fmt.Sprintf("(Line %v: %v) ", p.CodeSourceMap[p.CodePointer]+1, tk.LimitString(p.Source[p.CodeSourceMap[p.CodePointer]], 50))+formatA, argsA...)
}

func (p *QuixieVM) ParamsToStrs(runA *RunningContext, v *Instr, fromA int) []string {
	if runA == nil {
		runA = p.Running
	}

	lenT := len(v.Params)

	sl := make([]string, 0, lenT)

	for i := fromA; i < lenT; i++ {
		sl = append(sl, tk.ToStr(p.GetVarValue(runA, v.Params[i])))
	}

	return sl
}

func (p *QuixieVM) ParamsToList(runA *RunningContext, v *Instr, fromA int) []interface{} {
	if runA == nil {
		runA = p.Running
	}

	lenT := len(v.Params)

	sl := make([]interface{}, 0, lenT)

	for i := fromA; i < lenT; i++ {
		sl = append(sl, p.GetVarValue(runA, v.Params[i]))
	}

	return sl
}

func RunInstr(p *QuixieVM, r *RunningContext, instrA *Instr) (resultR interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// tk.Printfln("exception: %v", r)

			resultR = fmt.Errorf("runtime exception: %v\n%v", r, string(debug.Stack()))

			return
		}
	}()

	var instrT *Instr

	instrT = instrA

	// if p.VerbosePlusM {
	// tk.Plv(instrT)
	// }

	cmdT := instrT.Code

	switch cmdT {
	case 12: // invalidInstr
		return p.Errf(r, "invalid instr: %v", instrT.Params[0].Value)
	case 100: // version
		var pr interface{} = -5

		if instrT.ParamLen > 0 {
			pr = instrT.Params[0]
		}

		p.SetVar(r, pr, VersionG)

		return ""
	case 101: // pass
		return ""

	case 103: // debugInfo
		var pr any = -5
		if instrT.ParamLen > 0 {
			pr = instrT.Params[0]
		}

		outT := map[string]interface{}{
			"VM":                    p,
			"CurrentRunningContext": r,
			"Instr":                 instrT,
		}

		p.SetVar(r, pr, tk.ToJSONX(outT, "-indent", "-sort"))

		return ""

	case 121: // test
		if instrT.ParamLen < 2 {
			return p.Errf(r, "not enough parameters")
		}

		v1 := p.GetVarValue(r, instrT.Params[0])
		v2 := p.GetVarValue(r, instrT.Params[1])

		var v3 string
		var v4 string

		if instrT.ParamLen > 3 {
			v3 = tk.ToStr(p.GetVarValue(r, instrT.Params[2]))
			v4 = "(" + tk.ToStr(p.GetVarValue(r, instrT.Params[3])) + ")"
		} else if instrT.ParamLen > 2 {
			v3 = tk.ToStr(p.GetVarValue(r, instrT.Params[2]))
		} else {
			v3 = tk.ToStr(GlobalsG.SyncSeq.Get())
		}

		if v1 == v2 {
			tk.Pl("test %v%v passed", v3, v4)
		} else {
			return p.Errf(r, "test %v%v failed: %#v <-> %#v", v3, v4, v1, v2)
		}

		return ""

	case 122: // testByStartsWith
		if instrT.ParamLen < 2 {
			return p.Errf(r, "not enough parameters")
		}

		v1 := p.GetVarValue(r, instrT.Params[0])
		v2 := p.GetVarValue(r, instrT.Params[1])

		var v3 string
		var v4 string

		if instrT.ParamLen > 3 {
			v3 = tk.ToStr(p.GetVarValue(r, instrT.Params[2]))
			v4 = "(" + tk.ToStr(p.GetVarValue(r, instrT.Params[3])) + ")"
		} else if instrT.ParamLen > 2 {
			v3 = tk.ToStr(p.GetVarValue(r, instrT.Params[2]))
		} else {
			v3 = tk.ToStr(GlobalsG.SyncSeq.Get())
		}

		if strings.HasPrefix(tk.ToStr(v1), tk.ToStr(v2)) {
			tk.Pl("test %v%v passed", v3, v4)
		} else {
			return p.Errf(r, "test %v%v failed: %#v <-> %#v", v3, v4, v1, v2)
		}

		return ""

	case 123: // testByReg
		if instrT.ParamLen < 2 {
			return p.Errf(r, "not enough parameters")
		}

		v1 := p.GetVarValue(r, instrT.Params[0])
		v2 := p.GetVarValue(r, instrT.Params[1])

		var v3 string
		var v4 string

		if instrT.ParamLen > 3 {
			v3 = tk.ToStr(p.GetVarValue(r, instrT.Params[2]))
			v4 = "(" + tk.ToStr(p.GetVarValue(r, instrT.Params[3])) + ")"
		} else if instrT.ParamLen > 2 {
			v3 = tk.ToStr(p.GetVarValue(r, instrT.Params[2]))
		} else {
			v3 = tk.ToStr(GlobalsG.SyncSeq.Get())
		}

		if tk.RegMatchX(tk.ToStr(v1), tk.ToStr(v2)) {
			tk.Pl("test %v%v passed", v3, v4)
		} else {
			return p.Errf(r, "test %v%v failed: %#v <-> %#v", v3, v4, v1, v2)
		}

		return ""

	case 153: // compile
		if instrT.ParamLen < 1 {
			return p.Errf(r, "not enough parameters")
		}

		var pr interface{} = -5
		v1p := 0

		if instrT.ParamLen > 1 {
			pr = instrT.Params[0]
			v1p = 1
		}

		rs := Compile(tk.ToStr(p.GetVarValue(r, instrT.Params[v1p])))

		p.SetVar(r, pr, rs)

		return ""

	case 180: // goto/jmp
		if instrT.ParamLen < 1 {
			return p.Errf(r, "not enough parameters")
		}

		v1 := p.GetVarValue(r, instrT.Params[0])

		c1 := r.GetLabelIndex(v1)

		if c1 >= 0 {
			return c1
		}

		// c, ok := v1.(int)

		// if ok {
		// 	return c
		// }

		// s2 := tk.ToStr(v1)

		// if len(s2) > 1 && s2[0:1] == ":" {
		// 	s2a := s2[1:]
		// 	if strings.HasPrefix(s2a, "+") {
		// 		return r.CodePointer + tk.ToInt(s2a[1:])
		// 	} else if strings.HasPrefix(s2, "-") {
		// 		return r.CodePointer - tk.ToInt(s2a[1:])
		// 	} else {
		// 		labelPointerT, ok := r.Labels[s2a]

		// 		if ok {
		// 			return labelPointerT
		// 		}
		// 	}
		// }

		return p.Errf(r, "invalid label: %v", v1)

	case 199: // exit
		if instrT.ParamLen < 1 {
			return "exit"
		}

		valueT := p.GetVarValue(r, instrT.Params[0])

		p.SetVarGlobal("outG", valueT)

		return "exit"

	case 201: // global
		if instrT.ParamLen < 1 {
			return p.Errf(r, "参数不够")
		}

		pr := instrT.Params[0]
		v1p := 1

		// contextT := p.CurrentFuncContextM

		if instrT.ParamLen < 2 {
			p.SetVarGlobal(pr, nil)
			// contextT.VarsM[nameT] = ""
			return ""
		}

		v1 := p.GetVarValue(r, instrT.Params[v1p])

		if v1 == "bool" {
			if instrT.ParamLen > 2 {
				p.SetVarGlobal(pr, tk.ToBool(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarGlobal(pr, false)
			}
		} else if v1 == "int" {
			if instrT.ParamLen > 2 {
				p.SetVarGlobal(pr, tk.ToInt(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarGlobal(pr, int(0))
			}
		} else if v1 == "byte" {
			if instrT.ParamLen > 2 {
				p.SetVarGlobal(pr, tk.ToByte(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarGlobal(pr, byte(0))
			}
		} else if v1 == "rune" {
			if instrT.ParamLen > 2 {
				p.SetVarGlobal(pr, tk.ToRune(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarGlobal(pr, rune(0))
			}
		} else if v1 == "float" {
			if instrT.ParamLen > 2 {
				p.SetVarGlobal(pr, tk.ToFloat(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarGlobal(pr, float64(0.0))
			}
		} else if v1 == "str" {
			if instrT.ParamLen > 2 {
				p.SetVarGlobal(pr, tk.ToStr(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarGlobal(pr, "")
			}
		} else if v1 == "list" || v1 == "array" || v1 == "[]" {
			blT := make([]interface{}, 0, instrT.ParamLen-2)

			vs := p.ParamsToList(r, instrT, v1p+1)

			for _, vvv := range vs {
				nv, ok := vvv.([]interface{})

				if ok {
					for _, vvvj := range nv {
						blT = append(blT, vvvj)
					}
				} else {
					blT = append(blT, vvv)
				}
			}

			p.SetVarGlobal(pr, blT)
		} else if v1 == "strList" {
			blT := make([]string, 0, instrT.ParamLen-2)

			vs := p.ParamsToList(r, instrT, v1p+1)

			for _, vvv := range vs {
				nv, ok := vvv.([]string)

				if ok {
					for _, vvvj := range nv {
						blT = append(blT, vvvj)
					}
				} else {
					blT = append(blT, tk.ToStr(vvv))
				}
			}

			p.SetVarGlobal(pr, blT)
		} else if v1 == "byteList" {
			blT := make([]byte, 0, instrT.ParamLen-2)

			vs := p.ParamsToList(r, instrT, v1p+1)

			for _, vvv := range vs {
				nv, ok := vvv.([]byte)

				if ok {
					for _, vvvj := range nv {
						blT = append(blT, vvvj)
					}
				} else {
					blT = append(blT, tk.ToByte(vvv, 0))
				}
			}

			p.SetVarGlobal(pr, blT)
		} else if v1 == "runeList" {
			blT := make([]rune, 0, instrT.ParamLen-2)

			vs := p.ParamsToList(r, instrT, v1p+1)

			for _, vvv := range vs {
				nv, ok := vvv.([]rune)

				if ok {
					for _, vvvj := range nv {
						blT = append(blT, vvvj)
					}
				} else {
					blT = append(blT, tk.ToRune(vvv, 0))
				}
			}

			p.SetVarGlobal(pr, blT)
		} else if v1 == "map" {
			p.SetVarGlobal(pr, map[string]interface{}{})
		} else if v1 == "strMap" {
			p.SetVarGlobal(pr, map[string]string{})
		} else if v1 == "time" || v1 == "time.Time" {
			if instrT.ParamLen > 2 {
				p.SetVarGlobal(pr, tk.ToTime(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarGlobal(pr, time.Now())
			}
		} else {
			switch v1 {
			case "gui":
				objT := p.GetVarValue(r, VarRef{Ref: 3, Value: "guiG"})
				p.SetVarGlobal(pr, objT)
			case "quickDelegate":
				if instrT.ParamLen < 2 {
					return p.Errf(r, "not enough parameters")
				}

				v2 := r.GetLabelIndex(p.GetVarValue(r, instrT.Params[v1p+1]))

				var deleT tk.QuickDelegate

				// same as fastCall, to be modified!!!
				deleT = func(strA string) string {
					pointerT := r.CodePointer

					p.Stack.Push(strA)

					tmpPointerT := v2

					for {
						rs := RunInstr(p, r, &r.InstrList[tmpPointerT])

						nv, ok := rs.(int)

						if ok {
							tmpPointerT = nv
							continue
						}

						nsv, ok := rs.(string)

						if ok {
							if tk.IsErrStr(nsv) {
								// tmpRs := p.Pop()
								r.CodePointer = pointerT
								return nsv
							}

							if nsv == "exit" { // 不应发生
								tmpRs := p.Stack.Pop()
								r.CodePointer = pointerT
								return tk.ToStr(tmpRs)
							} else if nsv == "fr" {
								break
							}
						}

						tmpPointerT++
					}

					// return pointerT + 1

					tmpRs := p.Stack.Pop()
					r.CodePointer = pointerT
					return tk.ToStr(tmpRs)
				}

				p.SetVarGlobal(pr, deleT)
			case "image.Point", "point":
				var p1 image.Point
				if instrT.ParamLen > 3 {
					p1 = image.Point{X: tk.ToInt(p.GetVarValue(r, instrT.Params[2])), Y: tk.ToInt(p.GetVarValue(r, instrT.Params[3]))}
					p.SetVarGlobal(pr, p1)
				} else {
					p.SetVarGlobal(pr, p1)
				}
			default:
				p.SetVarGlobal(pr, nil)

			}

		}

		return ""

	case 203: // var
		if instrT.ParamLen < 1 {
			return p.Errf(r, "参数不够")
		}

		pr := instrT.Params[0]
		v1p := 1

		// contextT := p.CurrentFuncContextM

		if instrT.ParamLen < 2 {
			p.SetVarLocal(r, pr, nil)
			// contextT.VarsM[nameT] = ""
			return ""
		}

		v1 := p.GetVarValue(r, instrT.Params[v1p])

		if v1 == "bool" {
			if instrT.ParamLen > 2 {
				p.SetVarLocal(r, pr, tk.ToBool(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarLocal(r, pr, false)
			}
		} else if v1 == "int" {
			if instrT.ParamLen > 2 {
				p.SetVarLocal(r, pr, tk.ToInt(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarLocal(r, pr, int(0))
			}
		} else if v1 == "byte" {
			if instrT.ParamLen > 2 {
				p.SetVarLocal(r, pr, tk.ToByte(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarLocal(r, pr, byte(0))
			}
		} else if v1 == "rune" {
			if instrT.ParamLen > 2 {
				p.SetVarLocal(r, pr, tk.ToRune(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarLocal(r, pr, rune(0))
			}
		} else if v1 == "float" {
			if instrT.ParamLen > 2 {
				p.SetVarLocal(r, pr, tk.ToFloat(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarLocal(r, pr, float64(0.0))
			}
		} else if v1 == "str" {
			if instrT.ParamLen > 2 {
				p.SetVarLocal(r, pr, tk.ToStr(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarLocal(r, pr, "")
			}
		} else if v1 == "list" || v1 == "array" || v1 == "[]" {
			blT := make([]interface{}, 0, instrT.ParamLen-2)

			vs := p.ParamsToList(r, instrT, v1p+1)

			for _, vvv := range vs {
				nv, ok := vvv.([]interface{})

				if ok {
					for _, vvvj := range nv {
						blT = append(blT, vvvj)
					}
				} else {
					blT = append(blT, vvv)
				}
			}

			p.SetVarLocal(r, pr, blT)
		} else if v1 == "strList" {
			blT := make([]string, 0, instrT.ParamLen-2)

			vs := p.ParamsToList(r, instrT, v1p+1)

			for _, vvv := range vs {
				nv, ok := vvv.([]string)

				if ok {
					for _, vvvj := range nv {
						blT = append(blT, vvvj)
					}
				} else {
					blT = append(blT, tk.ToStr(vvv))
				}
			}

			p.SetVarLocal(r, pr, blT)
		} else if v1 == "byteList" {
			blT := make([]byte, 0, instrT.ParamLen-2)

			vs := p.ParamsToList(r, instrT, v1p+1)

			for _, vvv := range vs {
				nv, ok := vvv.([]byte)

				if ok {
					for _, vvvj := range nv {
						blT = append(blT, vvvj)
					}
				} else {
					blT = append(blT, tk.ToByte(vvv, 0))
				}
			}

			p.SetVarLocal(r, pr, blT)
		} else if v1 == "runeList" {
			blT := make([]rune, 0, instrT.ParamLen-2)

			vs := p.ParamsToList(r, instrT, v1p+1)

			for _, vvv := range vs {
				nv, ok := vvv.([]rune)

				if ok {
					for _, vvvj := range nv {
						blT = append(blT, vvvj)
					}
				} else {
					blT = append(blT, tk.ToRune(vvv, 0))
				}
			}

			p.SetVarLocal(r, pr, blT)
		} else if v1 == "map" {
			p.SetVarLocal(r, pr, map[string]interface{}{})
		} else if v1 == "strMap" {
			p.SetVarLocal(r, pr, map[string]string{})
		} else if v1 == "time" || v1 == "time.Time" {
			if instrT.ParamLen > 2 {
				p.SetVarLocal(r, pr, tk.ToTime(p.GetVarValue(r, instrT.Params[2])))
			} else {
				p.SetVarLocal(r, pr, time.Now())
			}
		} else {
			switch v1 {
			case "gui":
				objT := p.GetVarValue(r, VarRef{Ref: 3, Value: "guiG"})
				p.SetVarLocal(r, pr, objT)
			case "quickDelegate":
				if instrT.ParamLen < 2 {
					return p.Errf(r, "not enough parameters")
				}

				v2 := r.GetLabelIndex(p.GetVarValue(r, instrT.Params[v1p+1]))

				var deleT tk.QuickDelegate

				// same as fastCall, to be modified!!!
				deleT = func(strA string) string {
					pointerT := r.CodePointer

					p.Stack.Push(strA)

					tmpPointerT := v2

					for {
						rs := RunInstr(p, r, &r.InstrList[tmpPointerT])

						nv, ok := rs.(int)

						if ok {
							tmpPointerT = nv
							continue
						}

						nsv, ok := rs.(string)

						if ok {
							if tk.IsErrStr(nsv) {
								// tmpRs := p.Pop()
								r.CodePointer = pointerT
								return nsv
							}

							if nsv == "exit" { // 不应发生
								tmpRs := p.Stack.Pop()
								r.CodePointer = pointerT
								return tk.ToStr(tmpRs)
							} else if nsv == "fr" {
								break
							}
						}

						tmpPointerT++
					}

					// return pointerT + 1

					tmpRs := p.Stack.Pop()
					r.CodePointer = pointerT
					return tk.ToStr(tmpRs)
				}

				p.SetVarLocal(r, pr, deleT)
			case "image.Point", "point":
				var p1 image.Point
				if instrT.ParamLen > 3 {
					p1 = image.Point{X: tk.ToInt(p.GetVarValue(r, instrT.Params[2])), Y: tk.ToInt(p.GetVarValue(r, instrT.Params[3]))}
					p.SetVarLocal(r, pr, p1)
				} else {
					p.SetVarLocal(r, pr, p1)
				}
			default:
				p.SetVarLocal(r, pr, nil)

			}

		}

		return ""

	case 220: // push
		if instrT.ParamLen < 1 {
			return p.Errf(r, "not enough parameters")
		}

		if instrT.ParamLen > 1 {
			v2 := p.GetVarValue(r, instrT.Params[1])

			v1 := tk.ToStr(p.GetVarValue(r, instrT.Params[0]))

			if v1 == "int" {
				p.Stack.Push(tk.ToInt(v2))
			} else if v1 == "byte" {
				p.Stack.Push(tk.ToByte(v2))
			} else if v1 == "rune" {
				p.Stack.Push(tk.ToRune(v2))
			} else if v1 == "float" {
				p.Stack.Push(tk.ToFloat(v2))
			} else if v1 == "bool" {
				p.Stack.Push(tk.ToBool(v2))
			} else if v1 == "str" {
				p.Stack.Push(tk.ToStr(v2))
			} else {
				p.Stack.Push(v2)
			}

			return ""
		}

		v1 := p.GetVarValue(r, instrT.Params[0])

		p.Stack.Push(v1)

		return ""
	case 222: // peek
		if instrT.ParamLen < 1 {
			return p.Errf(r, "not enough parameters")
		}

		pr := instrT.Params[0]

		errT := p.SetVar(r, pr, p.Stack.Peek())

		if errT != nil {
			return p.Errf(r, "%v", errT)
		}

		return ""

	case 224: // pop
		var pr interface{} = -5
		if instrT.ParamLen < 1 {
			p.SetVar(r, pr, p.Stack.Pop())
			return ""
		}

		pr = instrT.Params[0]

		p.SetVar(r, pr, p.Stack.Pop())

		return ""
	case 230: // getStackSize
		var pr interface{} = -5
		if instrT.ParamLen < 1 {
			p.SetVar(r, pr, p.Stack.Size())
			return ""
		}

		pr = instrT.Params[0]

		p.SetVar(r, pr, p.Stack.Size())

		return ""
	case 240: // clearStack
		p.Stack.Clear()
		return ""

	case 401: // assign/=
		if instrT.ParamLen < 2 {
			return p.Errf(r, "not enough parameters")
		}

		pr := instrT.Params[0]

		if instrT.ParamLen > 2 {
			valueTypeT := instrT.Params[1].Value
			valueT := p.GetVarValue(r, instrT.Params[2])

			if valueTypeT == "bool" {
				p.SetVar(r, pr, tk.ToBool(valueT))
			} else if valueTypeT == "int" {
				p.SetVar(r, pr, tk.ToInt(valueT))
			} else if valueTypeT == "byte" {
				p.SetVar(r, pr, tk.ToByte(valueT))
			} else if valueTypeT == "rune" {
				p.SetVar(r, pr, tk.ToRune(valueT))
			} else if valueTypeT == "float" {
				p.SetVar(r, pr, tk.ToFloat(valueT))
			} else if valueTypeT == "str" {
				p.SetVar(r, pr, tk.ToStr(valueT))
			} else if valueTypeT == "list" || valueT == "array" || valueT == "[]" {
				p.SetVar(r, pr, valueT.([]interface{}))
			} else if valueTypeT == "strList" {
				p.SetVar(r, pr, valueT.([]string))
			} else if valueTypeT == "byteList" {
				p.SetVar(r, pr, valueT.([]byte))
			} else if valueTypeT == "runeList" {
				p.SetVar(r, pr, valueT.([]rune))
			} else if valueTypeT == "map" {
				p.SetVar(r, pr, valueT.(map[string]interface{}))
			} else if valueTypeT == "strMap" {
				p.SetVar(r, pr, valueT.(map[string]string))
			} else if valueTypeT == "time" {
				p.SetVar(r, pr, valueT.(map[string]string))
			} else {
				p.SetVar(r, pr, valueT)
			}

			return ""
		}

		valueT := p.GetVarValue(r, instrT.Params[1])

		p.SetVar(r, pr, valueT)

		// (*(p.CurrentVarsM))[nameT] = valueT

		return ""

	case 801: // inc
		if instrT.ParamLen < 1 {
			v1 := p.Stack.Pop()

			nv, ok := v1.(int)

			if ok {
				p.Stack.Push(nv + 1)
				return ""
			}

			nv2, ok := v1.(byte)

			if ok {
				p.Stack.Push(nv2 + 1)
				return ""
			}

			nv3, ok := v1.(rune)

			if ok {
				p.Stack.Push(nv3 + 1)
				return ""
			}

			p.Stack.Push(tk.ToInt(v1) + 1)

			return ""
		}

		v1 := p.GetVarValue(r, instrT.Params[0])

		nv, ok := v1.(int)

		if ok {
			p.SetVar(r, instrT.Params[0], nv+1)
			return ""
		}

		nv2, ok := v1.(byte)

		if ok {
			p.SetVar(r, instrT.Params[0], nv2+1)
			return ""
		}

		nv3, ok := v1.(rune)

		if ok {
			p.SetVar(r, instrT.Params[0], nv3+1)
			return ""
		}

		p.SetVar(r, instrT.Params[0], tk.ToInt(v1)+1)

		return ""

	case 810: // dec
		if instrT.ParamLen < 1 {
			v1 := p.Stack.Pop()

			nv, ok := v1.(int)

			if ok {
				p.Stack.Push(nv - 1)
				return ""
			}

			nv2, ok := v1.(byte)

			if ok {
				p.Stack.Push(nv2 - 1)
				return ""
			}

			nv3, ok := v1.(rune)

			if ok {
				p.Stack.Push(nv3 - 1)
				return ""
			}

			p.Stack.Push(tk.ToInt(v1) - 1)

			return ""
		}

		v1 := p.GetVarValue(r, instrT.Params[0])

		nv, ok := v1.(int)

		if ok {
			p.SetVar(r, instrT.Params[0], nv-1)
			return ""
		}

		nv2, ok := v1.(byte)

		if ok {
			p.SetVar(r, instrT.Params[0], nv2-1)
			return ""
		}

		nv3, ok := v1.(rune)

		if ok {
			p.SetVar(r, instrT.Params[0], nv3-1)
			return ""
		}

		p.SetVar(r, instrT.Params[0], tk.ToInt(v1)-1)

		return ""

	case 901: // add
		// tk.Plv(instrT)
		if instrT.ParamLen < 2 {
			return p.Errf(r, "not enough parameters")
		}

		var pr interface{} = -5
		v1p := 0

		if instrT.ParamLen > 2 {
			pr = instrT.Params[0]
			v1p = 1
		}

		v1 := p.GetVarValue(r, instrT.Params[v1p])
		v2 := p.GetVarValue(r, instrT.Params[v1p+1])

		var v3 interface{}

		switch nv := v1.(type) {
		case int:
			v3 = nv + v2.(int)
		case byte:
			v3 = nv + v2.(byte)
		case rune:
			v3 = nv + v2.(rune)
		case float64:
			v3 = nv + v2.(float64)
		case string:
			v3 = nv + v2.(string)
		case time.Time:
			v3 = nv.Add(time.Duration(time.Millisecond * time.Duration(tk.ToInt(v2))))
		default:
			return p.Errf(r, "type not match: %T(%v) -> %T(%v)", v1, v1, v2, v2)
		}

		p.SetVar(r, pr, v3)

		return ""

	case 1010: // call
		if instrT.ParamLen < 1 {
			return p.Errf(r, "not enough paramters")
		}

		v1 := p.GetVarValue(r, instrT.Params[0])

		v1c := r.GetLabelIndex(v1)

		if v1c < 0 {
			return p.Errf(r, "invalid label format: %v", v1)
		}

		r.PointerStack.Push(r.CodePointer)

		r.PushFunc()

		return v1c

	case 1020: // ret
		rsi := r.RunDefer(p)

		if tk.IsErrX(rsi) {
			return p.Errf(r, "[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStrX(rsi))
		}

		errT := r.PopFunc()

		if errT != nil {
			return p.Errf(r, "failed to return from function call: %v", errT)
		}

		newPointT := r.PointerStack.Pop()

		if newPointT == nil || tk.IsUndefined(newPointT) {
			return p.Errf(r, "no return pointer from function call: %v", newPointT)
		}

		return tk.ToInt(newPointT) + 1
	case 1055: // sealCall
		if instrT.ParamLen < 2 {
			return p.Errf(r, "not enough parameters")
		}

		pr := instrT.Params[0]
		v1p := 1

		codeT := p.GetVarValue(r, instrT.Params[v1p])

		vs := p.ParamsToList(r, instrT, v1p+1)

		if s1, ok := codeT.(string); ok {
			s1 = strings.ReplaceAll(s1, "~~~", "`")

			compiledT := Compile(s1)

			if tk.IsErrX(compiledT) {
				p.SetVar(r, pr, compiledT)
				return ""
			}

			codeT = compiledT
		}

		if cp1, ok := codeT.(*CompiledCode); ok {
			rs := p.RunCompiledCode(cp1, vs)

			p.SetVar(r, pr, rs)
			return ""
		}

		p.SetVar(r, pr, fmt.Errorf("failed to compile code: %v", codeT))
		return ""

		// if instrT.ParamLen > 1 {
		// 	argCountT = tk.ToInt(p.GetVarValue(r, instrT.Params[0]))

		// 	codeT = p.GetVarValue(r, instrT.Params[1]).(string)
		// } else {
		// 	codeT = p.GetVarValue(r, instrT.Params[0]).(string)
		// }

		// codeT = strings.ReplaceAll(codeT, "~~~", "`")

		// return p.CallFunc(codeT, argCountT, vs...)

	case 1070: // fastCall
		if instrT.ParamLen < 1 {
			return p.Errf(r, "not enough parameters")
		}

		v1 := p.GetVarValue(r, instrT.Params[0])

		c1 := r.GetLabelIndex(v1)

		if c1 < 0 {
			return p.Errf(r, "invalid label: %v", v1)
		}

		r.PointerStack.Push(r.CodePointer)

		return c1

	case 1071: // fastRet
		rs := r.PointerStack.Pop()

		if tk.IsUndefined(rs) {
			return p.Errf(r, "pointer stack empty")
		}

		return rs.(int) + 1

	case 1123: // getArrayItem/[]
		if instrT.ParamLen < 3 {
			return p.Errf(r, "not enough parameters")
		}

		var pr = instrT.Params[0]

		v1 := p.GetVarValue(r, instrT.Params[1])

		// tk.Pl("v1: %#v, instr: %#v", v1, instrT)

		v2 := tk.ToInt(p.GetVarValue(r, instrT.Params[2]))

		switch nv := v1.(type) {
		case []interface{}:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}
			p.SetVar(r, pr, nv[v2])
		case []bool:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}
			p.SetVar(r, pr, nv[v2])
		case []int:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}
			p.SetVar(r, pr, nv[v2])
		case []byte:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}
			p.SetVar(r, pr, nv[v2])
		case []rune:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}
			p.SetVar(r, pr, nv[v2])
		case []int64:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}
			p.SetVar(r, pr, nv[v2])
		case []float64:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}
			p.SetVar(r, pr, nv[v2])
		case []string:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}
			p.SetVar(r, pr, nv[v2])
		case []map[string]string:
			if (v2 < 0) || (v2 >= len(nv)) {
				if instrT.ParamLen > 3 {
					p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
					return ""
				} else {
					return p.Errf(r, "index out of range: %v/%v", v2, len(nv))
				}
			}

			p.SetVar(r, pr, nv[v2])
		default:
			if instrT.ParamLen > 3 {
				p.SetVar(r, pr, p.GetVarValue(r, instrT.Params[3]))
			} else {
				p.SetVar(r, pr, tk.Undefined)
			}
			return p.Errf(r, "parameter types not match: %#v", instrT)
		}

		return ""

	case 10410: // pln
		list1T := []interface{}{}

		for _, v := range instrT.Params {
			list1T = append(list1T, p.GetVarValue(r, v))
		}

		fmt.Println(list1T...)

		return ""

	case 10430: // plv
		if instrT.ParamLen < 1 {
			tk.Plv(p.GetCurrentFuncContext(r).Tmp)
			return ""
		}

		s1 := p.GetVarValue(r, instrT.Params[0])

		tk.Plv(s1)

		return ""

	case 20601: // systemCmd
		if instrT.ParamLen < 2 {
			return p.Errf(r, "not enough parameters")
		}

		pr := instrT.Params[0]
		v1p := 1

		v1 := tk.ToStr(p.GetVarValue(r, instrT.Params[v1p]))

		optsA := p.ParamsToStrs(r, instrT, v1p+1)

		// tk.Pln(v1, ",", optsA)

		p.SetVar(r, pr, tk.SystemCmd(v1, optsA...))

		return ""

		// end of switch

		//// ------ cmd end

	}

	return p.Errf(r, "unknown instr: %v", instrT)
}

func (p *QuixieVM) Run(posA ...int) interface{} {
	// tk.Pl("%#v", p)
	p.Running.CodePointer = 0
	if len(posA) > 0 {
		p.Running.CodePointer = posA[0]
	}

	if len(p.Running.CodeList) < 1 {
		return tk.Undefined
	}

	for {
		// tk.Pl("-- [%v] %v", p.CodePointerM, tk.LimitString(p.SourceM[p.CodeSourceMapM[p.CodePointerM]], 50))
		resultT := RunInstr(p, p.Running, &p.Running.InstrList[p.Running.CodePointer])

		c1T, ok := resultT.(int)

		if ok {
			p.Running.CodePointer = c1T
		} else {
			if tk.IsErrX(resultT) {
				p.RunDeferUpToRoot(p.Running)
				return p.Errf(p.Running, "[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStrX(resultT))
				// tk.Pl("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), p.CodeSourceMapM[p.CodePointerM]+1, tk.GetErrStr(rs))
				// break
			}

			rs, ok := resultT.(string)

			if !ok {
				p.RunDeferUpToRoot(p.Running)
				return p.Errf(p.Running, "return result error: (%T)%v", resultT, resultT)
			}

			if tk.IsErrStrX(rs) {
				p.RunDeferUpToRoot(p.Running)
				return p.Errf(p.Running, "[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStr(rs))
				// tk.Pl("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), p.CodeSourceMapM[p.CodePointerM]+1, tk.GetErrStr(rs))
				// break
			}

			if rs == "" {
				p.Running.CodePointer++

				if p.Running.CodePointer >= len(p.Running.CodeList) {
					break
				}
			} else if rs == "exit" {

				break
				// } else if rs == "cont" {
				// 	return p.Errf(r, "无效指令: %v", rs)
				// } else if rs == "brk" {
				// 	return p.Errf(r, "无效指令: %v", rs)
			} else {
				tmpI := tk.StrToInt(rs)

				if tmpI < 0 {
					p.RunDeferUpToRoot(p.Running)

					return p.Errf(p.Running, "invalid instr: %v", rs)
				}

				if tmpI >= len(p.Running.CodeList) {
					p.RunDeferUpToRoot(p.Running)
					return p.Errf(p.Running, "instr index out of range: %v(%v)/%v", tmpI, rs, len(p.Running.CodeList))
				}

				p.Running.CodePointer = tmpI
			}

		}

	}

	rsi := p.RunDeferUpToRoot(p.Running)

	if tk.IsErrX(rsi) {
		return tk.ErrStrf("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStrX(rsi))
	}

	// tk.Pl(tk.ToJSONX(p, "-indent", "-sort"))

	outT, ok := p.GetFuncContext(p.Running, 0).Vars["outG"]
	if !ok {
		return tk.Undefined
	}

	return outT

}

func (p *QuixieVM) RunCompiledCode(codeA *CompiledCode, inputA interface{}) interface{} {
	r := NewRunningContext(codeA)

	if tk.IsErrX(r) {
		return r
	}

	rp := r.(*RunningContext)

	if len(rp.CodeList) < 1 {
		return tk.Undefined
	}

	rp.CodePointer = 0

	rp.PushFunc()

	func1 := rp.FuncStack.Peek().(*FuncContext)
	func1.Vars["inputL"] = inputA

	for {
		// tk.Pl("-- [%v] %v", p.CodePointerM, tk.LimitString(p.SourceM[p.CodeSourceMapM[p.CodePointerM]], 50))
		resultT := RunInstr(p, rp, &rp.InstrList[rp.CodePointer])

		c1T, ok := resultT.(int)

		if ok {
			rp.CodePointer = c1T
		} else {
			if tk.IsErrX(resultT) {
				rp.RunDeferUpToRoot(p)
				return p.Errf(rp, "[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStrX(resultT))
				// tk.Pl("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), p.CodeSourceMapM[p.CodePointerM]+1, tk.GetErrStr(rs))
				// break
			}

			rs, ok := resultT.(string)

			if !ok {
				rp.RunDeferUpToRoot(p)
				return p.Errf(rp, "return result error: (%T)%v", resultT, resultT)
			}

			if tk.IsErrStrX(rs) {
				rp.RunDeferUpToRoot(p)
				return p.Errf(rp, "[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStr(rs))
				// tk.Pl("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), p.CodeSourceMapM[p.CodePointerM]+1, tk.GetErrStr(rs))
				// break
			}

			if rs == "" {
				rp.CodePointer++

				if rp.CodePointer >= len(rp.CodeList) {
					break
				}
			} else if rs == "exit" {

				break
				// } else if rs == "cont" {
				// 	return p.Errf(r, "无效指令: %v", rs)
				// } else if rs == "brk" {
				// 	return p.Errf(r, "无效指令: %v", rs)
			} else {
				tmpI := tk.StrToInt(rs)

				if tmpI < 0 {
					rp.RunDeferUpToRoot(p)

					return p.Errf(rp, "invalid instr: %v", rs)
				}

				if tmpI >= len(rp.CodeList) {
					rp.RunDeferUpToRoot(p)
					return p.Errf(rp, "instr index out of range: %v(%v)/%v", tmpI, rs, len(rp.CodeList))
				}

				rp.CodePointer = tmpI
			}

		}

	}

	rsi := rp.RunDeferUpToRoot(p)

	if tk.IsErrX(rsi) {
		return tk.ErrStrf("[%v](xie) runtime error: %v", tk.GetNowTimeStringFormal(), tk.GetErrStrX(rsi))
	}

	// tk.Pl(tk.ToJSONX(p, "-indent", "-sort"))

	outT, ok := func1.Vars["outL"]
	if !ok {
		return tk.Undefined
	}

	return outT
}

func RunCode(codeA string, objA interface{}, optsA ...string) interface{} {
	vmT := NewVMQuick()

	if len(optsA) > 0 {
		vmT.SetVar(vmT.Running, "argsG", optsA)
	}

	objT, ok := objA.(map[string]interface{})

	if ok {
		for k, v := range objT {
			vmT.SetVar(vmT.Running, k, v)
		}
	} else {
		if objA != nil {
			vmT.SetVar(vmT.Running, "inputG", objA)
		}
	}

	lrs := vmT.Load(vmT.Running, codeA)

	if tk.IsErrX(lrs) {
		return lrs
	}

	rs := vmT.Run()

	return rs
}

func (p *QuixieVM) Debug() {
	tk.Pln(tk.ToJSONX(p, "-indent", "-sort"))
}

var GlobalsG *GlobalContext

func init() {
	// tk.Pl("init")

	InstrCodeSet = make(map[int]string, 0)

	for k, v := range InstrNameSet {
		InstrCodeSet[v] = k
	}

	GlobalsG = &GlobalContext{}

	GlobalsG.Vars = make(map[string]interface{}, 0)

	GlobalsG.Vars["backQuote"] = "`"
}
