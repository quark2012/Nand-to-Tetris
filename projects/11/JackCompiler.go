package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	jackExtension   = ".jack"
	outputExtension = ".vm"
	identationSize  = 2
)

const (
	TOKEN_TYPE_KEYWORD = iota
	TOKEN_TYPE_SYMBOL
	TOKEN_TYPE_IDENTIFIER
	TOKEN_TYPE_INT_CONST
	TOKEN_TYPE_STR_CONST
)

const (
	IDENTIFIER_CATEGORY_VAR = iota
	IDENTIFIER_CATEGORY_ARGUMENT
	IDENTIFIER_CATEGORY_STATIC
	IDENTIFIER_CATEGORY_FIELD
	IDENTIFIER_CATEGORY_CLASS
	IDENTIFIER_CATEGORY_SUBROUTINE
)

const (
	IDENTIFIER_BEING_DEFINED = iota
	IDENTIFIER_BEING_USED
)

const (
	SYMBOL_KIND_STATIC = iota
	SYMBOL_KIND_FIELD
	SYMBOL_KIND_ARG
	SYMBOL_KIND_VAR
	SYMBOL_KIND_NONE
)

const (
	SEGMENT_STATIC = iota
	SEGMENT_THIS
	SEGMENT_ARGUMENT
	SEGMENT_LOCAL
	SEGMENT_CONSTANT
	SEGMENT_THAT
	SEGMENT_POINTER
	SEGMENT_TEMP
)

const (
	COMMAND_ADD = iota
	COMMAND_SUB
	COMMAND_NEG
	COMMAND_EQ
	COMMAND_GT
	COMMAND_LT
	COMMAND_AND
	COMMAND_OR
	COMMAND_NOT
)

const (
	VM_COMMAND_TYPE_LABEL = iota
	VM_COMMAND_TYPE_GOTO
	VM_COMMAND_TYPE_IF
	VM_COMMAND_TYPE_CALL
	VM_COMMAND_TYPE_FUNCTION
	VM_COMMAND_TYPE_RETURN
)

var (
	integerRe                = regexp.MustCompile(`^(\d|[1-9]\d|[1-9]\d{2}|[1-9]\d{3}|[1-2]\d{4}|3[0-1]\d{3}|32[0-6]\d{2}|327[0-5]\d|3276[0-7])$`)
	identifierRe             = regexp.MustCompile(`^([a-zA-Z_]\w+|[a-zA-Z_])$`)
	identifierCategoryLookup = map[int]string{
		IDENTIFIER_CATEGORY_VAR:        "var",
		IDENTIFIER_CATEGORY_ARGUMENT:   "argument",
		IDENTIFIER_CATEGORY_STATIC:     "static",
		IDENTIFIER_CATEGORY_FIELD:      "field",
		IDENTIFIER_CATEGORY_CLASS:      "class",
		IDENTIFIER_CATEGORY_SUBROUTINE: "subroutine",
	}
	identifierBeingLookup = map[int]string{
		IDENTIFIER_BEING_DEFINED: "defined",
		IDENTIFIER_BEING_USED:    "used",
	}
	symbolKindLookup = map[int]int{
		SYMBOL_KIND_STATIC: SEGMENT_STATIC,
		SYMBOL_KIND_FIELD:  SEGMENT_THIS,
		SYMBOL_KIND_ARG:    SEGMENT_ARGUMENT,
		SYMBOL_KIND_VAR:    SEGMENT_LOCAL,
	}
)

func showErr(str string) {
	log.Fatalln("Error:", str)
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln("Error:", err)
	}
}

type Token struct {
	tokenType int
	value     string
}

type Tokenizer struct {
	fileName     string
	stream       string
	currentToken *Token
	nextToken    *Token
}

func NewTokenizer(fileName string) Tokenizer {
	o := Tokenizer{fileName: fileName}

	// fmt.Println("processing file", fileName)
	data, err := ioutil.ReadFile(fileName)
	checkErr(err)

	// remove comments
	re := regexp.MustCompile("(?s)//.*?\n|/\\*.*?\\*/")
	o.stream = string(re.ReplaceAll(data, nil))

	o.process()

	return o
}

func (t Tokenizer) hasMoreTokens() bool {
	return t.nextToken != nil
}

func (t *Tokenizer) advance() {
	t.currentToken = t.nextToken
	t.nextToken = nil
	t.process()
}

func (t Tokenizer) matchSymbol(c rune) bool {
	return c == '{' || c == '}' || c == '(' || c == ')' || c == '[' || c == ']' ||
		c == '.' || c == ',' || c == ';' ||
		c == '+' || c == '-' || c == '*' || c == '/' || c == '&' || c == '|' ||
		c == '<' || c == '>' || c == '=' || c == '~'
}

func (t Tokenizer) matchKeyword(s string) bool {
	return s == "class" || s == "method" || s == "function" || s == "constructor" ||
		s == "int" || s == "boolean" || s == "char" || s == "void" ||
		s == "var" || s == "static" || s == "field" ||
		s == "let" || s == "do" || s == "if" || s == "else" || s == "while" || s == "return" ||
		s == "true" || s == "false" || s == "null" || s == "this"
}

func (t Tokenizer) debug(i int, rune rune, buildStr bool, found bool, stream string) {
	// log.Printf("%d: [%c][%d][str=%v][found=%v][%v]\n", i, rune, rune, buildStr, found, stream)
}

func (t *Tokenizer) process() {
	var b bytes.Buffer
	var found, buildStr bool = false, false
	var index, tokenType int = -1, -1

	for i, rune := range t.stream {
		index = i

		// building a string - A sequence of characters not including double quote or newline
		if buildStr {
			// double quote and newline will terminate this
			if rune == '"' || rune == '\n' || rune == '\r' {
				buildStr = false
				tokenType = TOKEN_TYPE_STR_CONST
				found = true
				t.debug(i, rune, buildStr, found, b.String())
				break
			} else {
				b.WriteRune(rune)
				t.debug(i, rune, buildStr, found, b.String())
				continue
			}
		}

		// double quote
		if rune == '"' {
			buildStr = true
			t.debug(i, rune, buildStr, found, b.String())
			continue
		}

		// symbol and no built up string
		if t.matchSymbol(rune) && b.Len() == 0 {
			tokenType = TOKEN_TYPE_SYMBOL
			found = true
			b.WriteRune(rune)
			t.debug(i, rune, buildStr, found, b.String())
			break
		}

		// one of the symbols, whitespaces, newlines
		if t.matchSymbol(rune) || rune == ' ' || rune == '\t' || rune == '\n' || rune == '\r' {
			// check the symbol again next round
			if t.matchSymbol(rune) {
				index -= 1
			}

			// is built up string a keyword
			if t.matchKeyword(b.String()) {
				tokenType = TOKEN_TYPE_KEYWORD
				found = true
				t.debug(i, rune, buildStr, found, b.String())
				break
			}

			// is built up string an integer
			if integerRe.MatchString(b.String()) {
				tokenType = TOKEN_TYPE_INT_CONST
				found = true
				t.debug(i, rune, buildStr, found, b.String())
				break
			}

			// is built up string an identifier
			if identifierRe.MatchString(b.String()) {
				tokenType = TOKEN_TYPE_IDENTIFIER
				found = true
				t.debug(i, rune, buildStr, found, b.String())
				break
			}

			// an error
			if b.Len() > 0 {
				log.Fatalln("Error: unknown string element", b.String())
				// log.Printf("b:\t[%v]\n", []byte(b.String()))
				b.Reset()
			}

			// skip whitespaces and newlines
			if rune == ' ' || rune == '\t' || rune == '\n' || rune == '\r' {
				continue
			}
		}

		// build up the string
		b.WriteRune(rune)
		t.debug(i, rune, buildStr, found, b.String())
	}

	// store the rest of the stream
	t.stream = t.stream[index+1:]

	// store the next token
	if found {
		t.nextToken = &Token{tokenType: tokenType, value: b.String()}
	}

	// log.Printf("[index=%d][str=%v][found=%v][token=%v][%d]\n", index, buildStr, found, t.nextToken, len(t.stream))
}

func (t Tokenizer) tokenType() int {
	return t.currentToken.tokenType
}

func (t Tokenizer) keyword() string {
	return t.currentToken.value
}

func (t Tokenizer) symbol() string {
	// switch t.currentToken.value {
	// case "<":
	// 	return "&lt;"
	// case ">":
	// 	return "&gt;"
	// case "\"":
	// 	return "&quot;"
	// case "&":
	// 	return "&amp;"
	// default:
	// 	return t.currentToken.value
	// }
	return t.currentToken.value
}

func (t Tokenizer) identifier() string {
	return t.currentToken.value
}

func (t Tokenizer) intVal() int {
	val, err := strconv.Atoi(t.currentToken.value)
	if err != nil {
		return -1
	} else {
		return val
	}
}

func (t Tokenizer) stringVal() string {
	return t.currentToken.value
}

type CompilationEngine struct {
	tokenizer        Tokenizer
	classSymbolTable SymbolTable
	subSymbolTable   SymbolTable
	vmWriter         VMWriter
	labelCount       int
	className        string
	subroutineName   string
	varName          string
	currentFuncType  string
	currentType      string

	currentIdent int
	identStr     string
	outputStr    bytes.Buffer
}

func NewCompilationEngine(inputFile string) CompilationEngine {
	name := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	dst := filepath.Join(filepath.Dir(inputFile), name+outputExtension)

	o := CompilationEngine{
		tokenizer:        NewTokenizer(inputFile),
		classSymbolTable: NewSymbolTable(),
		subSymbolTable:   NewSymbolTable(),
		vmWriter:         NewVMWriter(dst),
		labelCount:       0,
	}

	o.classSymbolTable.reset()

	return o
}

func (ce *CompilationEngine) writeStartTag(tag string) {
	str := fmt.Sprintf("%s<%s>\n", ce.identStr, tag)
	ce.outputStr.WriteString(str)
	ce.changeIdent(identationSize)
	// log.Println("writeStartTag", strings.TrimSuffix(str, "\n"))
}

func (ce *CompilationEngine) writeEndTag(tag string) {
	ce.changeIdent(-identationSize)
	str := fmt.Sprintf("%s</%s>\n", ce.identStr, tag)
	ce.outputStr.WriteString(str)
	// log.Println("writeEndTag", strings.TrimSuffix(str, "\n"))
}

func (ce *CompilationEngine) writeFullTag(tag string, value string) {
	str := fmt.Sprintf("%s<%s> %s </%s>\n", ce.identStr, tag, value, tag)
	ce.outputStr.WriteString(str)
	// log.Println("writeFullTag", strings.TrimSuffix(str, "\n"))
}

func (ce *CompilationEngine) changeIdent(change int) {
	ce.currentIdent += change
	ce.identStr = strings.Repeat(" ", ce.currentIdent)
}

func (ce *CompilationEngine) nextToken() {
	if ce.tokenizer.hasMoreTokens() {
		ce.tokenizer.advance()
	} else {
		ce.tokenizer.currentToken = nil
		// showErr("no more tokens")
	}
}

func (ce *CompilationEngine) expectBase(tokenType int, match string, optional bool) bool {
	token := ce.tokenizer.currentToken
	// log.Println("tokenType:", tokenType, "token.tokenType:", token.tokenType, "match:", match, "token.value:", token.value, "optional:", optional)
	if token.tokenType == tokenType && (tokenType == TOKEN_TYPE_INT_CONST ||
		tokenType == TOKEN_TYPE_STR_CONST || tokenType == TOKEN_TYPE_IDENTIFIER) {
		if !optional {
			switch tokenType {
			case TOKEN_TYPE_INT_CONST:
				ce.writeFullTag("integerConstant", ce.tokenizer.stringVal())
			case TOKEN_TYPE_STR_CONST:
				ce.writeFullTag("stringConstant", ce.tokenizer.stringVal())
			case TOKEN_TYPE_IDENTIFIER:
				ce.writeFullTag("identifier", ce.tokenizer.identifier())
			default:
				showErr("shouldn't reach here")
			}
			ce.nextToken()
		}
		return true
	}
	if token.tokenType == tokenType && (tokenType == TOKEN_TYPE_KEYWORD ||
		tokenType == TOKEN_TYPE_SYMBOL) && token.value == match {
		if !optional {
			switch tokenType {
			case TOKEN_TYPE_KEYWORD:
				ce.writeFullTag("keyword", ce.tokenizer.keyword())
			case TOKEN_TYPE_SYMBOL:
				ce.writeFullTag("symbol", ce.tokenizer.symbol())
			default:
				showErr("shouldn't reach here")
			}
			ce.nextToken()
		}
		return true
	} else {
		if !optional {
			switch tokenType {
			case TOKEN_TYPE_INT_CONST:
				showErr("missing integerConstant")
			case TOKEN_TYPE_STR_CONST:
				showErr("missing stringConstant")
			case TOKEN_TYPE_IDENTIFIER:
				showErr("missing identifier " + match)
			case TOKEN_TYPE_KEYWORD:
				showErr("missing keyword " + match)
			case TOKEN_TYPE_SYMBOL:
				showErr("missing symbol " + match)
			default:
				showErr("shouldn't reach here")
			}
		}
		return false
	}
}

func (ce *CompilationEngine) expectIntConstant(optional bool) bool {
	return ce.expectBase(TOKEN_TYPE_INT_CONST, "", optional)
}

func (ce *CompilationEngine) expectStrConstant(optional bool) bool {
	return ce.expectBase(TOKEN_TYPE_STR_CONST, "", optional)
}

func (ce *CompilationEngine) expectKeyword(keyword string, optional bool) bool {
	return ce.expectBase(TOKEN_TYPE_KEYWORD, keyword, optional)
}

func (ce *CompilationEngine) expectIdentifier(identifier string, optional bool, context int, being int) bool {
	token := ce.tokenizer.currentToken
	// log.Println("expectIdentifier", "identifier:", identifier, "token.value:", token.value, "optional:", optional)
	if token.tokenType == TOKEN_TYPE_IDENTIFIER {
		if !optional {
			if context == IDENTIFIER_CATEGORY_CLASS {
				ce.className = ce.tokenizer.identifier()
				// log.Println("expectIdentifier", "className", ce.className)
			} else if context == IDENTIFIER_CATEGORY_SUBROUTINE {
				ce.subroutineName = ce.tokenizer.identifier()
				// log.Println("expectIdentifier", "subroutineName", ce.subroutineName)
			} else if context == IDENTIFIER_CATEGORY_STATIC {
				ce.classSymbolTable.define(ce.tokenizer.identifier(), ce.currentType, SYMBOL_KIND_STATIC)
				ce.currentType = ""
			} else if context == IDENTIFIER_CATEGORY_FIELD {
				ce.classSymbolTable.define(ce.tokenizer.identifier(), ce.currentType, SYMBOL_KIND_FIELD)
				ce.currentType = ""
			} else if context == IDENTIFIER_CATEGORY_ARGUMENT {
				ce.subSymbolTable.define(ce.tokenizer.identifier(), ce.currentType, SYMBOL_KIND_ARG)
				ce.currentType = ""
			} else if context == IDENTIFIER_CATEGORY_VAR {
				if being == IDENTIFIER_BEING_DEFINED {
					ce.subSymbolTable.define(ce.tokenizer.identifier(), ce.currentType, SYMBOL_KIND_VAR)
					ce.currentType = ""
				} else if being == IDENTIFIER_BEING_USED {
					ce.varName = ce.tokenizer.identifier()
					// log.Println("expectIdentifier", "varName", ce.varName)
				}
			}

			ce.writeFullTag("identifier", ce.tokenizer.identifier())
			ce.nextToken()
		}
		return true
	} else {
		if !optional {
			showErr("missing identifier " + identifier)
		}
		return false
	}
}

func (ce *CompilationEngine) expectSymbol(symbol string, optional bool) bool {
	return ce.expectBase(TOKEN_TYPE_SYMBOL, symbol, optional)
}

func (ce *CompilationEngine) peekKeywords(keywords ...string) bool {
	for _, keyword := range keywords {
		if ce.expectKeyword(keyword, true) {
			return true
		}
	}
	return false
}

func (ce *CompilationEngine) peekSymbols(symbols ...string) bool {
	for _, symbol := range symbols {
		if ce.expectSymbol(symbol, true) {
			return true
		}
	}
	return false
}

func (ce *CompilationEngine) findVarName(varName string) (bool, int, string, int) {
	var kind, index int
	var stype string
	// log.Println("varName", varName, "classSymbolTable", ce.classSymbolTable, "subSymbolTable", ce.subSymbolTable)
	kind = ce.subSymbolTable.kindOf(varName)
	if kind == SYMBOL_KIND_NONE {
		kind = ce.classSymbolTable.kindOf(varName)
		if kind == SYMBOL_KIND_NONE {
			return false, kind, stype, index
		} else {
			stype = ce.classSymbolTable.typeOf(varName)
			index = ce.classSymbolTable.indexOf(varName)
		}
	} else {
		stype = ce.subSymbolTable.typeOf(varName)
		index = ce.subSymbolTable.indexOf(varName)
	}
	return true, symbolKindLookup[kind], stype, index
}

// type: 'int' | 'char'| 'boolean' | className
// ('void' | type)
func (ce *CompilationEngine) expectType(includeVoid bool, optional bool) bool {
	if ce.expectIdentifier("className", true, -1, -1) { // identifier className
		if !optional {
			ce.currentType = ce.tokenizer.identifier()
			// log.Println("expectType", "currentType", ce.currentType)
			ce.expectIdentifier("className", false, -1, -1)
		}
		return true
	} else if ce.peekKeywords("int", "char", "boolean") || (includeVoid && ce.peekKeywords("void")) {
		if !optional {
			ce.currentType = ce.tokenizer.keyword()
			// log.Println("expectType", "currentType", ce.currentType)
			ce.writeFullTag("keyword", ce.tokenizer.keyword())
			ce.nextToken()
		}

		return true
	} else {
		return false
	}
}

// classVarDec: ('static'|'field') type varName (',' varName)* ';'
// varName: identifier
func (ce *CompilationEngine) expectClassVarDec(optional bool) bool {
	// keywords ('static'|'field')
	if ce.peekKeywords("static", "field") {
		if !optional {
			keyword := ce.tokenizer.keyword()
			var context int
			if keyword == "static" {
				context = IDENTIFIER_CATEGORY_STATIC
			} else {
				context = IDENTIFIER_CATEGORY_FIELD
			}

			ce.writeFullTag("keyword", ce.tokenizer.keyword())
			ce.nextToken()

			ce.expectType(false, false)
			ce.expectIdentifier("varName", false, context, IDENTIFIER_BEING_DEFINED)

			// (',' varName)*
			for {
				if ce.expectSymbol(",", true) {
					ce.expectSymbol(",", false)
					ce.expectIdentifier("varName", false, context, IDENTIFIER_BEING_DEFINED)
				} else {
					break
				}
			}

			ce.expectSymbol(";", false)
		}

		return true
	} else {
		return false
	}
}

// subroutineDec: ('constructor'| 'function'| 'method') ('void' | type) subroutineName '(' parameterList ')' subroutineBody
// subroutineName: identifier
func (ce *CompilationEngine) expectSubroutine(optional bool) bool {
	// keywords ('constructor'| 'function'| 'method')
	if ce.peekKeywords("constructor", "function", "method") {
		if !optional {
			ce.subSymbolTable.reset()

			ce.currentFuncType = ce.tokenizer.keyword()
			// log.Println("expectSubroutine", "currentFuncType", ce.currentFuncType)
			if ce.currentFuncType == "method" {
				ce.subSymbolTable.define("this", ce.className, SYMBOL_KIND_ARG)
			}

			ce.writeFullTag("keyword", ce.tokenizer.keyword())
			ce.nextToken()

			ce.expectType(true, false) // ('void' | type)
			ce.expectIdentifier("subroutineName", false, IDENTIFIER_CATEGORY_SUBROUTINE, IDENTIFIER_BEING_DEFINED)
			ce.expectSymbol("(", false)
			ce.compileParameterList()
			ce.expectSymbol(")", false)
			ce.compileSubroutineBody()
		}

		return true
	} else {
		return false
	}
}

// varDec: 'var' type varName (',' varName)* ';'
func (ce *CompilationEngine) expectVarDec(optional bool) bool {
	// keyword 'var'
	if ce.expectKeyword("var", true) {
		if !optional {
			ce.expectKeyword("var", false)
			ce.expectType(false, false)

			varType := ce.currentType

			ce.expectIdentifier("varName", false, IDENTIFIER_CATEGORY_VAR, IDENTIFIER_BEING_DEFINED)

			// (',' varName)*
			for {
				if ce.expectSymbol(",", true) {
					ce.expectSymbol(",", false)

					ce.currentType = varType
					// log.Println("expectVarDec", "currentType", ce.currentType)

					ce.expectIdentifier("varName", false, IDENTIFIER_CATEGORY_VAR, IDENTIFIER_BEING_DEFINED)
				} else {
					break
				}
			}

			ce.expectSymbol(";", false)
		}

		return true
	} else {
		return false
	}
}

// statement: letStatement | ifStatement | whileStatement | doStatement | returnStatement
func (ce *CompilationEngine) peekStatement() bool {
	// keywords 'let' | 'if' | 'while' | 'do' | 'return'
	if ce.peekKeywords("let", "if", "while", "do", "return") {
		return true
	} else {
		return false
	}
}

// term: integerConstant | stringConstant | keywordConstant | varName | varName '[' expression ']' | '(' expression ')' | (unaryOp term) | subroutineCall
// subroutineCall: subroutineName '(' expressionList ')' | (className | varName) '.' subroutineName '(' expressionList ')'
// unaryOp: '-' | '~'
// integerConstant: A decimal integer in the range 0...32767
// stringConstant: '"' A sequence of characters not including double quote or newline '"'
// keywordConstant: 'true' | 'false' | 'null' | 'this'
func (ce *CompilationEngine) expectTerm(optional bool) bool {
	if ce.expectIntConstant(true) {
		if !optional {
			constant := ce.tokenizer.intVal()

			ce.expectIntConstant(false)

			// vm code: push constant <integer>
			ce.vmWriter.writePush(SEGMENT_CONSTANT, constant)
		}
	} else if ce.expectStrConstant(true) {
		if !optional {
			str := ce.tokenizer.stringVal()

			ce.expectStrConstant(false)

			// vm code: push constant <string length>
			// vm code: call String.new 1
			// vm code: push constant <character c>
			// vm code: call String.appendChar 2
			ce.vmWriter.writeCreateString(str)
		}
	} else if ce.peekKeywords("true", "false", "null", "this") {
		// keywordConstant: 'true' | 'false' | 'null' | 'this'
		if !optional {
			keyword := ce.tokenizer.keyword()

			ce.writeFullTag("keyword", ce.tokenizer.keyword())
			ce.nextToken()

			if keyword == "true" {
				// vm code: push constant 1
				// vm code: neg
				ce.vmWriter.writePush(SEGMENT_CONSTANT, 1)
				ce.vmWriter.writeArithmetic(COMMAND_NEG)
			} else if keyword == "false" || keyword == "null" {
				// vm code: push constant 0
				ce.vmWriter.writePush(SEGMENT_CONSTANT, 0)
			} else if keyword == "this" {
				// vm code: push pointer 0
				ce.vmWriter.writePush(SEGMENT_POINTER, 0)
			}
		}
	} else if ce.expectSymbol("(", true) {
		// '(' expression ')'
		if !optional {
			ce.expectSymbol("(", false)
			ce.compileExpression()
			ce.expectSymbol(")", false)
		}
	} else if ce.expectSymbol("-", true) || ce.expectSymbol("~", true) {
		// (unaryOp term)
		// unaryOp: '-' | '~'
		if !optional {
			symbol := ce.tokenizer.symbol()
			ce.writeFullTag("symbol", ce.tokenizer.symbol())
			ce.nextToken()
			ce.compileTerm()
			if symbol == "-" {
				// vm code: neg
				ce.vmWriter.writeArithmetic(COMMAND_NEG)
			} else if symbol == "~" {
				// vm code: not
				ce.vmWriter.writeArithmetic(COMMAND_NOT)
			}
		}
	} else if ce.expectIdentifier("classNameOrVarNameOrSubroutineName", true, IDENTIFIER_CATEGORY_VAR, IDENTIFIER_BEING_USED) {
		// varName | varName '[' expression ']' | subroutineCall
		// subroutineCall: subroutineName '(' expressionList ')' | (className | varName) '.' subroutineName '(' expressionList ')'
		if !optional {
			// varName | subroutineName | className
			ce.expectIdentifier("classNameOrVarNameOrSubroutineName", false, IDENTIFIER_CATEGORY_VAR, IDENTIFIER_BEING_USED)
			varName := ce.varName
			ce.varName = ""

			// '[' | '(' | '.' | nothing
			if ce.expectSymbol("[", true) {
				// varName '[' expression ']'

				// vm code: push varName[expression]
				found, segment, _, index := ce.findVarName(varName)
				// log.Println("expectTerm", "varName", varName, "found:", found, "segment:", segment, "stype:", stype, "index:", index)
				if found {
					// varName
					// vm code: push segment index
					ce.vmWriter.writePush(segment, index)
				} else {
					showErr("cannot find varName " + varName)
				}

				ce.expectSymbol("[", false)
				ce.compileExpression()
				ce.expectSymbol("]", false)

				// vm code: add
				// vm code: pop pointer 1
				// vm code: push that 0
				ce.vmWriter.writeArithmetic(COMMAND_ADD)
				ce.vmWriter.writePop(SEGMENT_POINTER, 1)
				ce.vmWriter.writePush(SEGMENT_THAT, 0)
			} else if ce.expectSymbol("(", true) {
				// subroutineName '(' expressionList ')'

				// vm code: push pointer 0
				ce.vmWriter.writePush(SEGMENT_POINTER, 0)

				ce.expectSymbol("(", false)
				n := ce.compileExpressionList()
				ce.expectSymbol(")", false)

				// vm code: call this.varName n+1
				funcName := fmt.Sprintf("%s.%s", ce.className, varName)
				ce.vmWriter.writeCall(funcName, n+1)
			} else if ce.expectSymbol(".", true) {
				// (className | varName) '.' subroutineName '(' expressionList ')'

				ce.expectSymbol(".", false)
				ce.expectIdentifier("subroutineName", false, IDENTIFIER_CATEGORY_SUBROUTINE, -1)
				subroutineName := ce.subroutineName
				ce.subroutineName = ""

				found, segment, stype, index := ce.findVarName(varName)
				// log.Println("expectTerm", "varName", varName, "found:", found, "segment:", segment, "stype:", stype, "index:", index)
				if found {
					// varName
					// vm code: push segment index
					ce.vmWriter.writePush(segment, index)
				} else {
					// className
				}

				ce.expectSymbol("(", false)
				n := ce.compileExpressionList()
				ce.expectSymbol(")", false)

				if found {
					// varName
					// vm code: call stype.subroutineName n+1
					funcName := fmt.Sprintf("%s.%s", stype, subroutineName)
					ce.vmWriter.writeCall(funcName, n+1)
				} else {
					// className
					// vm code: call className.subroutineName n
					funcName := fmt.Sprintf("%s.%s", varName, subroutineName)
					ce.vmWriter.writeCall(funcName, n)
				}
			} else {
				// vm code: push varName
				found, segment, _, index := ce.findVarName(varName)
				// log.Println("expectTerm", "varName", varName, "found:", found, "segment:", segment, "stype:", stype, "index:", index)
				if found {
					// varName
					// vm code: push segment index
					ce.vmWriter.writePush(segment, index)
				} else {
					showErr("cannot find varName " + varName)
				}
			}
		}
	} else {
		return false
	}
	return true
}

// class: 'class' className '{' classVarDec* subroutineDec* '}'
// className: identifier
// identifier: A sequence of letters, digits, and underscore ('_'), not starting with a digit
// tag: class
func (ce *CompilationEngine) compileClass() {
	ce.writeStartTag("class")

	ce.nextToken()

	ce.expectKeyword("class", false)
	ce.expectIdentifier("className", false, IDENTIFIER_CATEGORY_CLASS, IDENTIFIER_BEING_DEFINED)
	ce.expectSymbol("{", false)
	ce.compileClassVarDec() // classVarDec*
	ce.compileSubroutine()  // subroutineDec*
	ce.expectSymbol("}", false)

	ce.writeEndTag("class")

	// err := ioutil.WriteFile(ce.outputFile, []byte(ce.outputStr.String()), 0644)
	// checkErr(err)
}

// classVarDec*
// tag: classVarDec
func (ce *CompilationEngine) compileClassVarDec() {
	for {
		if ce.expectClassVarDec(true) {
			ce.writeStartTag("classVarDec")
			ce.expectClassVarDec(false)
			ce.writeEndTag("classVarDec")
		} else {
			break
		}
	}
}

// subroutineDec*
// tag: subroutineDec
func (ce *CompilationEngine) compileSubroutine() {
	for {
		if ce.expectSubroutine(true) {
			ce.writeStartTag("subroutineDec")
			ce.expectSubroutine(false)
			ce.writeEndTag("subroutineDec")
		} else {
			break
		}
	}
}

// parameterList: ((type varName) (',' type varName)*)?
// tag: parameterList
func (ce *CompilationEngine) compileParameterList() {
	ce.writeStartTag("parameterList")

	// (type varName)?
	if ce.expectType(false, true) {
		ce.expectType(false, false)
		ce.expectIdentifier("varName", false, IDENTIFIER_CATEGORY_ARGUMENT, IDENTIFIER_BEING_DEFINED)

		// (',' type varName)*
		for {
			// symbol ,
			if ce.expectSymbol(",", true) {
				ce.expectSymbol(",", false)
				ce.expectType(false, false)
				ce.expectIdentifier("varName", false, IDENTIFIER_CATEGORY_ARGUMENT, IDENTIFIER_BEING_DEFINED)
			} else {
				break
			}
		}
	}

	ce.writeEndTag("parameterList")
}

// subroutineBody: '{' varDec* statements '}'
// tag: subroutineBody
func (ce *CompilationEngine) compileSubroutineBody() {
	ce.writeStartTag("subroutineBody")

	ce.expectSymbol("{", false)
	ce.compileVarDec() // varDec*

	funcName := fmt.Sprintf("%s.%s", ce.className, ce.subroutineName)
	ce.subroutineName = ""

	// vm code: function f nVars
	ce.vmWriter.writeFunction(funcName, ce.subSymbolTable.varCount(SYMBOL_KIND_VAR))

	if ce.currentFuncType == "constructor" {
		// vm code: push constant <number of fields>
		// vm code: call Memory.alloc 1
		// vm code: pop pointer 0
		ce.vmWriter.writePush(SEGMENT_CONSTANT, ce.classSymbolTable.varCount(SYMBOL_KIND_FIELD))
		ce.vmWriter.writeCall("Memory.alloc", 1)
		ce.vmWriter.writePop(SEGMENT_POINTER, 0)
	} else if ce.currentFuncType == "function" {
	} else if ce.currentFuncType == "method" {
		// vm code: push argument 0
		// vm code: pop pointer 0
		ce.vmWriter.writePush(SEGMENT_ARGUMENT, 0)
		ce.vmWriter.writePop(SEGMENT_POINTER, 0)
	}
	ce.currentFuncType = ""

	ce.compileStatements() // statements
	ce.expectSymbol("}", false)

	ce.writeEndTag("subroutineBody")
}

// varDec*
// tag: varDec
func (ce *CompilationEngine) compileVarDec() {
	for {
		if ce.expectVarDec(true) {
			ce.writeStartTag("varDec")
			ce.expectVarDec(false)
			ce.writeEndTag("varDec")
		} else {
			break
		}
	}
}

// statements: statement*
// tag: statements
func (ce *CompilationEngine) compileStatements() {
	ce.writeStartTag("statements")

	for {
		if ce.peekStatement() {
			// keywords 'let' | 'if' | 'while' | 'do' | 'return'
			if ce.expectKeyword("let", true) {
				ce.compileLet()
			} else if ce.expectKeyword("if", true) {
				ce.compileIf()
			} else if ce.expectKeyword("while", true) {
				ce.compileWhile()
			} else if ce.expectKeyword("do", true) {
				ce.compileDo()
			} else if ce.expectKeyword("return", true) {
				ce.compileReturn()
			}
		} else {
			break
		}
	}

	ce.writeEndTag("statements")
}

// letStatement: 'let' varName ('[' expression ']')? '=' expression ';'
// tag: letStatement
func (ce *CompilationEngine) compileLet() {
	arrayFound := false

	ce.writeStartTag("letStatement")

	ce.expectKeyword("let", false)
	ce.expectIdentifier("varName", false, IDENTIFIER_CATEGORY_VAR, IDENTIFIER_BEING_USED)
	varName := ce.varName
	ce.varName = ""

	found, segment, _, index := ce.findVarName(varName)
	// log.Println("compileLet", "varName", varName, "found:", found, "segment:", segment, "stype:", stype, "index:", index)
	if !found {
		showErr("cannot find varName " + varName)
	}

	// ('[' expression ']')?
	if ce.expectSymbol("[", true) {
		// let x[expression1] = expression2

		arrayFound = true

		// push varName[expression]
		// vm code: push segment index
		ce.vmWriter.writePush(segment, index)

		ce.expectSymbol("[", false)
		ce.compileExpression()
		ce.expectSymbol("]", false)

		// vm code: add
		ce.vmWriter.writeArithmetic(COMMAND_ADD)

	} else {
		// let x = expression
	}

	ce.expectSymbol("=", false)
	ce.compileExpression()
	ce.expectSymbol(";", false)

	if arrayFound {
		// let x[expression1] = expression2

		// vm code: pop temp 0
		// vm code: pop pointer 1
		// vm code: push temp 0
		// vm code: pop that 0
		ce.vmWriter.writePop(SEGMENT_TEMP, 0)
		ce.vmWriter.writePop(SEGMENT_POINTER, 1)
		ce.vmWriter.writePush(SEGMENT_TEMP, 0)
		ce.vmWriter.writePop(SEGMENT_THAT, 0)
	} else {
		// let x = expression

		// vm code: pop varName
		ce.vmWriter.writePop(segment, index)
	}

	ce.writeEndTag("letStatement")
}

// ifStatement: 'if' '(' expression ')' '{' statements '}' ('else' '{' statements '}')?
// tag: ifStatement
func (ce *CompilationEngine) compileIf() {
	ce.writeStartTag("ifStatement")

	ce.expectKeyword("if", false)
	ce.expectSymbol("(", false)
	ce.compileExpression()
	ce.expectSymbol(")", false)

	elseLabel := fmt.Sprintf("ifgoto_else_%d", ce.labelCount)
	ce.labelCount++
	endIfLabel := fmt.Sprintf("ifgoto_endif_%d", ce.labelCount)
	ce.labelCount++

	// vm code: not
	// vm code: if-goto L1
	ce.vmWriter.writeArithmetic(COMMAND_NOT)
	ce.vmWriter.writeIf(elseLabel)

	ce.expectSymbol("{", false)
	ce.compileStatements()
	ce.expectSymbol("}", false)

	// vm code: goto L2
	// vm code: label L1
	ce.vmWriter.writeGoto(endIfLabel)
	ce.vmWriter.writeLabel(elseLabel)

	// ('else' '{' statements '}')?
	if ce.expectKeyword("else", true) {
		ce.expectKeyword("else", false)
		ce.expectSymbol("{", false)
		ce.compileStatements()
		ce.expectSymbol("}", false)
	}

	// vm code: label L2
	ce.vmWriter.writeLabel(endIfLabel)

	ce.writeEndTag("ifStatement")
}

// whileStatement: 'while' '(' expression')' '{' statements '}'
// tag: whileStatement
func (ce *CompilationEngine) compileWhile() {
	ce.writeStartTag("whileStatement")

	whileLabel := fmt.Sprintf("ifgoto_while_%d", ce.labelCount)
	ce.labelCount++
	endWhileLabel := fmt.Sprintf("ifgoto_endwhile_%d", ce.labelCount)
	ce.labelCount++

	// vm code: label L1
	ce.vmWriter.writeLabel(whileLabel)

	ce.expectKeyword("while", false)
	ce.expectSymbol("(", false)
	ce.compileExpression()
	ce.expectSymbol(")", false)

	// vm code: not
	// vm code: if-goto L2
	ce.vmWriter.writeArithmetic(COMMAND_NOT)
	ce.vmWriter.writeIf(endWhileLabel)

	ce.expectSymbol("{", false)
	ce.compileStatements()
	ce.expectSymbol("}", false)

	// vm code: goto L1
	// vm code: label L2
	ce.vmWriter.writeGoto(whileLabel)
	ce.vmWriter.writeLabel(endWhileLabel)

	ce.writeEndTag("whileStatement")
}

// doStatement: 'do' subroutineCall ';'
// tag: doStatement
func (ce *CompilationEngine) compileDo() {
	ce.writeStartTag("doStatement")

	ce.expectKeyword("do", false)
	ce.expectTerm(false)
	ce.expectSymbol(";", false)

	// pop temp 0
	ce.vmWriter.writePop(SEGMENT_TEMP, 0)

	ce.writeEndTag("doStatement")
}

// returnStatement: 'return' expression? ';'
// tag: returnStatement
func (ce *CompilationEngine) compileReturn() {
	ce.writeStartTag("returnStatement")

	ce.expectKeyword("return", false)
	if ce.expectTerm(true) {
		ce.compileExpression()
	} else {
		// void
		// push constant 0
		ce.vmWriter.writePush(SEGMENT_CONSTANT, 0)
	}
	ce.expectSymbol(";", false)

	// return
	ce.vmWriter.writeReturn()

	ce.writeEndTag("returnStatement")
}

// expression: term (op term)*
// op: '+' | '-' | '*' | '/' | '&' | '|' | '<' | '>' | '='
// tag: expression
func (ce *CompilationEngine) compileExpression() {
	ce.writeStartTag("expression")

	ce.compileTerm()

	for {
		// op: '+' | '-' | '*' | '/' | '&' | '|' | '<' | '>' | '='
		if ce.peekSymbols("+", "-", "*", "/", "&", "|", "<", ">", "=") {
			symbol := ce.tokenizer.symbol()
			// log.Println("compileExpression", "symbol:", symbol)

			ce.writeFullTag("symbol", ce.tokenizer.symbol())
			ce.nextToken()
			ce.compileTerm()

			ce.vmWriter.writeOperator(symbol)
		} else {
			break
		}
	}

	ce.writeEndTag("expression")
}

// term: integerConstant | stringConstant | keywordConstant | varName | varName '[' expression ']' | '(' expression ')' | (unaryOp term) | subroutineCall
// subroutineCall: subroutineName '(' expressionList ')' | (className | varName) '.' subroutineName '(' expressionList ')'
// unaryOp: '-' | '~'
// integerConstant: A decimal integer in the range 0...32767
// stringConstant: '"' A sequence of characters not including double quote or newline '"'
// keywordConstant: 'true' | 'false' | 'null' | 'this'
// tag: term
func (ce *CompilationEngine) compileTerm() {
	ce.writeStartTag("term")
	ce.expectTerm(false)
	ce.writeEndTag("term")
}

// expressionList: (expression (',' expression)*)?
// tag: expressionList
func (ce *CompilationEngine) compileExpressionList() int {
	count := 0

	ce.writeStartTag("expressionList")

	// expression?
	if ce.expectTerm(true) {
		ce.compileExpression()
		count += 1

		// (',' expression)*
		for {
			if ce.expectSymbol(",", true) {
				ce.expectSymbol(",", false)
				ce.compileExpression()
				count += 1
			} else {
				break
			}
		}
	}

	ce.writeEndTag("expressionList")

	return count
}

type Table struct {
	name  string
	stype string
	kind  int
	index int
}

type SymbolTable struct {
	table   map[string]Table
	indexes [4]int
}

func NewSymbolTable() SymbolTable {
	o := SymbolTable{}
	// log.Println("NewSymbolTable", "SymbolTable", o)
	return o
}

func (st *SymbolTable) reset() {
	st.table = map[string]Table{}
	for i := range st.indexes {
		st.indexes[i] = 0
	}
	// log.Println("reset", "SymbolTable", st)
}

func (st *SymbolTable) define(name string, stype string, kind int) {
	index := st.indexes[kind]
	st.table[name] = Table{name: name, stype: stype, kind: kind, index: index}
	st.indexes[kind] += 1
	// log.Println("define", "SymbolTable", st)
}

func (st *SymbolTable) varCount(kind int) int {
	return st.indexes[kind]
}

func (st *SymbolTable) kindOf(name string) int {
	if item, ok := st.table[name]; ok {
		return item.kind
	} else {
		return SYMBOL_KIND_NONE
	}
}

func (st *SymbolTable) typeOf(name string) string {
	if item, ok := st.table[name]; ok {
		return item.stype
	} else {
		return ""
	}
}

func (st *SymbolTable) indexOf(name string) int {
	if item, ok := st.table[name]; ok {
		return item.index
	} else {
		return -1
	}
}

type VMWriter struct {
	outputFile       string
	buffer           bytes.Buffer
	segmentsLookup   map[int]string
	commandsLookup   map[int]string
	vmCommandsLookup map[int]string
}

func NewVMWriter(fileName string) VMWriter {
	o := VMWriter{outputFile: fileName, segmentsLookup: map[int]string{
		SEGMENT_CONSTANT: "constant",
		SEGMENT_ARGUMENT: "argument",
		SEGMENT_LOCAL:    "local",
		SEGMENT_STATIC:   "static",
		SEGMENT_THIS:     "this",
		SEGMENT_THAT:     "that",
		SEGMENT_POINTER:  "pointer",
		SEGMENT_TEMP:     "temp",
	}, commandsLookup: map[int]string{
		COMMAND_ADD: "add",
		COMMAND_SUB: "sub",
		COMMAND_NEG: "neg",
		COMMAND_EQ:  "eq",
		COMMAND_GT:  "gt",
		COMMAND_LT:  "lt",
		COMMAND_AND: "and",
		COMMAND_OR:  "or",
		COMMAND_NOT: "not",
	}, vmCommandsLookup: map[int]string{
		VM_COMMAND_TYPE_LABEL:    "label",
		VM_COMMAND_TYPE_GOTO:     "goto",
		VM_COMMAND_TYPE_IF:       "if-goto",
		VM_COMMAND_TYPE_CALL:     "call",
		VM_COMMAND_TYPE_FUNCTION: "function",
		VM_COMMAND_TYPE_RETURN:   "return",
	}}
	// log.Println("NewVMWriter", "VMWriter", o)
	return o
}

func (vw *VMWriter) writePush(segment int, index int) {
	str := fmt.Sprintf("push %s %d\n", vw.segmentsLookup[segment], index)
	vw.buffer.WriteString(str)
	// log.Println(strings.TrimSuffix(str, "\n"))
}

func (vw *VMWriter) writePop(segment int, index int) {
	str := fmt.Sprintf("pop %s %d\n", vw.segmentsLookup[segment], index)
	vw.buffer.WriteString(str)
	// log.Println(strings.TrimSuffix(str, "\n"))
}

func (vw *VMWriter) writeArithmetic(command int) {
	str := fmt.Sprintf("%s\n", vw.commandsLookup[command])
	vw.buffer.WriteString(str)
	// log.Println(strings.TrimSuffix(str, "\n"))
}

// VM_COMMAND_TYPE_LABEL:    "label",
// VM_COMMAND_TYPE_GOTO:     "goto",
// VM_COMMAND_TYPE_IF:       "if-goto",
// VM_COMMAND_TYPE_CALL:     "call",
// VM_COMMAND_TYPE_FUNCTION: "function",
// VM_COMMAND_TYPE_RETURN:   "return",

func (vw *VMWriter) writeVMCommand(vmCommandType int, label string, count int) {
	var str string
	switch vmCommandType {
	case VM_COMMAND_TYPE_LABEL, VM_COMMAND_TYPE_GOTO, VM_COMMAND_TYPE_IF:
		str = fmt.Sprintf("%s %s\n", vw.vmCommandsLookup[vmCommandType], label)
	case VM_COMMAND_TYPE_CALL, VM_COMMAND_TYPE_FUNCTION:
		str = fmt.Sprintf("%s %s %d\n", vw.vmCommandsLookup[vmCommandType], label, count)
	default:
		str = fmt.Sprintf("%s\n", vw.vmCommandsLookup[vmCommandType])
	}
	vw.buffer.WriteString(str)
	// log.Println(strings.TrimSuffix(str, "\n"))
}

func (vw *VMWriter) writeLabel(label string) {
	vw.writeVMCommand(VM_COMMAND_TYPE_LABEL, label, 0)
}

func (vw *VMWriter) writeGoto(label string) {
	vw.writeVMCommand(VM_COMMAND_TYPE_GOTO, label, 0)
}

func (vw *VMWriter) writeIf(label string) {
	vw.writeVMCommand(VM_COMMAND_TYPE_IF, label, 0)
}

func (vw *VMWriter) writeCall(label string, nArgs int) {
	vw.writeVMCommand(VM_COMMAND_TYPE_CALL, label, nArgs)
}

func (vw *VMWriter) writeFunction(label string, nVars int) {
	vw.writeVMCommand(VM_COMMAND_TYPE_FUNCTION, label, nVars)
}

func (vw *VMWriter) writeReturn() {
	vw.writeVMCommand(VM_COMMAND_TYPE_RETURN, "", 0)
}

func (vw *VMWriter) close() {
	err := ioutil.WriteFile(vw.outputFile, []byte(vw.buffer.String()), 0644)
	checkErr(err)
}

func (vw *VMWriter) writeCreateString(str string) {
	len := len(str)
	// vm code: push constant <string length>
	vw.writePush(SEGMENT_CONSTANT, len)
	// vm code: call String.new 1
	vw.writeCall("String.new", 1)
	for i := 0; i < len; i++ {
		// vm code: push constant <character c>
		vw.writePush(SEGMENT_CONSTANT, int(str[i]))
		// vm code: call String.appendChar 2
		vw.writeCall("String.appendChar", 2)
	}
}

// op: '+' | '-' | '*' | '/' | '&' | '|' | '<' | '>' | '='
func (vw *VMWriter) writeOperator(op string) {
	switch op {
	case "+":
		// vm code: add
		vw.writeArithmetic(COMMAND_ADD)
	case "-":
		// vm code: sub
		vw.writeArithmetic(COMMAND_SUB)
	case "*":
		// vm code: call Math.multiply 2
		vw.writeCall("Math.multiply", 2)
	case "/":
		// vm code: call Math.divide 2
		vw.writeCall("Math.divide", 2)
	case "&":
		// vm code: and
		vw.writeArithmetic(COMMAND_AND)
	case "|":
		// vm code: or
		vw.writeArithmetic(COMMAND_OR)
	case "<":
		// vm code: lt
		vw.writeArithmetic(COMMAND_LT)
	case ">":
		// vm code: gt
		vw.writeArithmetic(COMMAND_GT)
	case "=":
		// vm code: eq
		vw.writeArithmetic(COMMAND_EQ)
	default:
	}
}

func processFile(fileName string) {
	engine := NewCompilationEngine(fileName)
	engine.compileClass()
	engine.vmWriter.close()
}

func processDir(dirName string) {
	path := filepath.Join(dirName, "*"+jackExtension)
	files, err := filepath.Glob(path)
	checkErr(err)

	for _, file := range files {
		processFile(file)
	}
}

func main() {
	var file string

	args := os.Args
	if len(args) > 1 {
		file = args[1]
	} else {
		file = "."
	}

	info, err := os.Stat(file)
	checkErr(err)

	if info.IsDir() {
		processDir(file)
	} else {
		processFile(file)
	}
}
