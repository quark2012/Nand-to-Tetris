package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	outputHalf         = false
	jackExtension      = ".jack"
	xmlExtensionFull   = ".xmlf"
	xmlExtensionHalf   = ".xmlh"
	identationSize     = 2
	TOKEN_TYPE_KEYWORD = iota
	TOKEN_TYPE_SYMBOL
	TOKEN_TYPE_IDENTIFIER
	TOKEN_TYPE_INT_CONST
	TOKEN_TYPE_STR_CONST
)

var (
	integerRe    = regexp.MustCompile(`^(\d|[1-9]\d|[1-9]\d{2}|[1-9]\d{3}|[1-2]\d{4}|3[0-1]\d{3}|32[0-6]\d{2}|327[0-5]\d|3276[0-7])$`)
	identifierRe = regexp.MustCompile(`^([a-zA-Z_]\w+|[a-zA-Z_])$`)
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
	switch t.currentToken.value {
	case "<":
		return "&lt;"
	case ">":
		return "&gt;"
	case "\"":
		return "&quot;"
	case "&":
		return "&amp;"
	default:
		return t.currentToken.value
	}
}

func (t Tokenizer) identifier() string {
	return t.currentToken.value
}

func (t Tokenizer) intVal() string {
	return t.currentToken.value
}

func (t Tokenizer) stringVal() string {
	return t.currentToken.value
}

type CompilationEngine struct {
	outputFile   string
	tokenizer    Tokenizer
	currentIdent int
	identStr     string
	outputStr    bytes.Buffer
}

func NewCompilationEngine(inputFile string, outputFile string) CompilationEngine {
	o := CompilationEngine{outputFile: outputFile, tokenizer: NewTokenizer(inputFile)}
	return o
}

func (ce *CompilationEngine) writeStartTag(tag string) {
	str := fmt.Sprintf("%s<%s>\n", ce.identStr, tag)
	ce.outputStr.WriteString(str)
	ce.changeIdent(identationSize)
	// log.Println("Generated", str)
}

func (ce *CompilationEngine) writeEndTag(tag string) {
	ce.changeIdent(-identationSize)
	str := fmt.Sprintf("%s</%s>\n", ce.identStr, tag)
	ce.outputStr.WriteString(str)
	// log.Println("Generated", str)
}

func (ce *CompilationEngine) writeFullTag(tag string, value string) {
	str := fmt.Sprintf("%s<%s> %s </%s>\n", ce.identStr, tag, value, tag)
	ce.outputStr.WriteString(str)
	// log.Println("Generated", str)
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

// x
// peek, has x?, if yes, only one, else error
// x?
// peek, has x?, if yes, only one, else skip
// x* => loop x?
// peek, has x?, if yes, loop until no more, else skip
// x|y
// x? | y? | error

// expectxxx(), 0, 1 or more
// ? - min 0, max 1
// * - min 0, no max
// 1 - min 1, max 1
// return number found

// return true if found, else false
// if optional is true, don't do any processing
// if optional is false, process if found, show error if not found

func (ce *CompilationEngine) expectBase(tokenType int, match string, optional bool) bool {
	token := ce.tokenizer.currentToken
	// log.Println("tokenType:", tokenType, "token.tokenType:", token.tokenType, "match:", match, "token.value:", token.value, "optional:", optional)
	if token.tokenType == tokenType && (tokenType == TOKEN_TYPE_INT_CONST ||
		tokenType == TOKEN_TYPE_STR_CONST || tokenType == TOKEN_TYPE_IDENTIFIER) {
		if !optional {
			switch tokenType {
			case TOKEN_TYPE_INT_CONST:
				ce.writeFullTag("integerConstant", ce.tokenizer.intVal())
			case TOKEN_TYPE_STR_CONST:
				ce.writeFullTag("stringConstant", ce.tokenizer.intVal())
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

func (ce *CompilationEngine) expectIdentifier(identifier string, optional bool) bool {
	return ce.expectBase(TOKEN_TYPE_IDENTIFIER, identifier, optional)
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

// type: 'int' | 'char'| 'boolean' | className
// ('void' | type)
func (ce *CompilationEngine) expectType(includeVoid bool, optional bool) bool {
	if ce.expectIdentifier("className", true) { // identifier className
		if !optional {
			ce.expectIdentifier("className", false)
		}
		return true
	} else if ce.peekKeywords("int", "char", "boolean") || (includeVoid && ce.peekKeywords("void")) {
		if !optional {
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
			ce.writeFullTag("keyword", ce.tokenizer.keyword())
			ce.nextToken()

			ce.expectType(false, false)
			ce.expectIdentifier("varName", false)

			// (',' varName)*
			for {
				if ce.expectSymbol(",", true) {
					ce.expectSymbol(",", false)
					ce.expectIdentifier("varName", false)
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
			ce.writeFullTag("keyword", ce.tokenizer.keyword())
			ce.nextToken()

			ce.expectType(true, false) // ('void' | type)
			ce.expectIdentifier("subroutineName", false)
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
			ce.expectIdentifier("varName", false)

			// (',' varName)*
			for {
				if ce.expectSymbol(",", true) {
					ce.expectSymbol(",", false)
					ce.expectIdentifier("varName", false)
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
			ce.expectIntConstant(false)
		}
	} else if ce.expectStrConstant(true) {
		if !optional {
			ce.expectStrConstant(false)
		}
	} else if ce.peekKeywords("true", "false", "null", "this") {
		// keywordConstant: 'true' | 'false' | 'null' | 'this'
		if !optional {
			ce.writeFullTag("keyword", ce.tokenizer.keyword())
			ce.nextToken()
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
			ce.writeFullTag("symbol", ce.tokenizer.symbol())
			ce.nextToken()
			ce.compileTerm()
		}
	} else if ce.expectIdentifier("classNameOrVarNameOrSubroutineName", true) {
		// varName | varName '[' expression ']' | subroutineCall
		// subroutineCall: subroutineName '(' expressionList ')' | (className | varName) '.' subroutineName '(' expressionList ')'
		if !optional {
			// varName | subroutineName | className
			ce.expectIdentifier("classNameOrVarNameOrSubroutineName", false)

			// '[' | '(' | '.' | nothing
			if ce.expectSymbol("[", true) {
				// varName '[' expression ']'
				ce.expectSymbol("[", false)
				ce.compileExpression()
				ce.expectSymbol("]", false)
			} else if ce.expectSymbol("(", true) {
				// subroutineName '(' expressionList ')'
				ce.expectSymbol("(", false)
				ce.compileExpressionList()
				ce.expectSymbol(")", false)
			} else if ce.expectSymbol(".", true) {
				// (className | varName) '.' subroutineName '(' expressionList ')'
				ce.expectSymbol(".", false)
				ce.expectIdentifier("subroutineName", false)
				ce.expectSymbol("(", false)
				ce.compileExpressionList()
				ce.expectSymbol(")", false)
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
	ce.expectIdentifier("className", false)
	ce.expectSymbol("{", false)
	ce.compileClassVarDec() // classVarDec*
	ce.compileSubroutine()  // subroutineDec*
	ce.expectSymbol("}", false)

	ce.writeEndTag("class")

	err := ioutil.WriteFile(ce.outputFile, []byte(ce.outputStr.String()), 0644)
	checkErr(err)
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
		ce.expectIdentifier("varName", false)

		// (',' type varName)*
		for {
			// symbol ,
			if ce.expectSymbol(",", true) {
				ce.expectSymbol(",", false)
				ce.expectType(false, false)
				ce.expectIdentifier("varName", false)
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
	ce.compileVarDec()     // varDec*
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
	ce.writeStartTag("letStatement")

	ce.expectKeyword("let", false)
	ce.expectIdentifier("varName", false)

	// ('[' expression ']')?
	if ce.expectSymbol("[", true) {
		ce.expectSymbol("[", false)
		ce.compileExpression()
		ce.expectSymbol("]", false)
	}

	ce.expectSymbol("=", false)
	ce.compileExpression()
	ce.expectSymbol(";", false)

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
	ce.expectSymbol("{", false)
	ce.compileStatements()
	ce.expectSymbol("}", false)

	// ('else' '{' statements '}')?
	if ce.expectKeyword("else", true) {
		ce.expectKeyword("else", false)
		ce.expectSymbol("{", false)
		ce.compileStatements()
		ce.expectSymbol("}", false)
	}

	ce.writeEndTag("ifStatement")
}

// whileStatement: 'while' '(' expression')' '{' statements '}'
// tag: whileStatement
func (ce *CompilationEngine) compileWhile() {
	ce.writeStartTag("whileStatement")

	ce.expectKeyword("while", false)
	ce.expectSymbol("(", false)
	ce.compileExpression()
	ce.expectSymbol(")", false)
	ce.expectSymbol("{", false)
	ce.compileStatements()
	ce.expectSymbol("}", false)

	ce.writeEndTag("whileStatement")
}

// doStatement: 'do' subroutineCall ';'
// tag: doStatement
func (ce *CompilationEngine) compileDo() {
	ce.writeStartTag("doStatement")

	ce.expectKeyword("do", false)
	ce.expectTerm(false)
	ce.expectSymbol(";", false)

	ce.writeEndTag("doStatement")
}

// returnStatement: 'return' expression? ';'
// tag: returnStatement
func (ce *CompilationEngine) compileReturn() {
	ce.writeStartTag("returnStatement")

	ce.expectKeyword("return", false)
	if ce.expectTerm(true) {
		ce.compileExpression()
	}
	ce.expectSymbol(";", false)

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
			ce.writeFullTag("symbol", ce.tokenizer.symbol())
			ce.nextToken()
			ce.compileTerm()
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
	count := 1

	ce.writeStartTag("expressionList")

	// expression?
	if ce.expectTerm(true) {
		ce.compileExpression()

		// (',' expression)*
		for {
			if ce.expectSymbol(",", true) {
				count += 1
				ce.expectSymbol(",", false)
				ce.compileExpression()
			} else {
				break
			}
		}
	}

	ce.writeEndTag("expressionList")

	return count
}

func processFile(fileName string) {
	if outputHalf {
		processFileHalf(fileName)
	} else {
		processFileFull(fileName)
	}
}

func processFileFull(fileName string) {
	var name string = strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	var dst string = filepath.Join(filepath.Dir(fileName), name+xmlExtensionFull)
	engine := NewCompilationEngine(fileName, dst)
	engine.compileClass()
}

func processFileHalf(fileName string) {
	var name string = strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	var dst string = filepath.Join(filepath.Dir(fileName), name+xmlExtensionHalf)
	var b bytes.Buffer

	b.WriteString("<tokens>\n")

	tokenizer := NewTokenizer(fileName)
	for tokenizer.hasMoreTokens() {
		tokenizer.advance()
		tokenType := tokenizer.tokenType()
		switch tokenType {
		case TOKEN_TYPE_KEYWORD:
			b.WriteString("<keyword> " + tokenizer.keyword() + " </keyword>\n")
		case TOKEN_TYPE_SYMBOL:
			b.WriteString("<symbol> " + tokenizer.symbol() + " </symbol>\n")
		case TOKEN_TYPE_IDENTIFIER:
			b.WriteString("<identifier> " + tokenizer.identifier() + " </identifier>\n")
		case TOKEN_TYPE_INT_CONST:
			b.WriteString("<integerConstant> " + tokenizer.intVal() + " </integerConstant>\n")
		case TOKEN_TYPE_STR_CONST:
			b.WriteString("<stringConstant> " + tokenizer.stringVal() + " </stringConstant>\n")
		default:
			log.Fatalln("Error: unknown token type", tokenType)
		}

		// break
	}

	b.WriteString("</tokens>\n")

	err := ioutil.WriteFile(dst, []byte(b.String()), 0644)
	checkErr(err)
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
