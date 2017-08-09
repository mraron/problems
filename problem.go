package problems

import (
	"io"
	"github.com/mraron/language"
	"time"
	"database/sql/driver"
	"encoding/json"
	"bytes"
	"errors"
)

type Content struct {
	Locale   string
	Contents []byte
	Type     string
}

func (s Content) IsText() bool {
	return s.Type == "text"
}

func (s Content) IsHTML() bool {
	return s.Type == "text/html"
}

func (s Content) IsPDF() bool {
	return s.Type == "application/pdf"
}

func (s Content) String() string {
	return string(s.Contents)
}

type VerdictName int

const (
	VERDICT_AC VerdictName = iota
	VERDICT_WA
	VERDICT_RE
	VERDICT_TL
	VERDICT_ML
	VERDICT_XX
)

func (v VerdictName) String() string {
	switch v {
	case VERDICT_AC:
		return "Elfogadva"
	case VERDICT_WA:
		return "Rossz válasz"
	case VERDICT_RE:
		return "Futási	 hiba"
	case VERDICT_TL:
		return "Időlimit túllépés"
	case VERDICT_ML:
		return "Memória limit túllépés"
	case VERDICT_XX:
		return "Belső hiba"
	}

	return "..."
}


type FeedbackType int

const (
	FEEDBACK_CF FeedbackType = iota
	FEEDBACK_IOI
	FEEDBACK_ACM
)

func FeedbackFromString(str string) FeedbackType {
	if str == "ioi" {
		return FEEDBACK_IOI
	}else if str == "acm" {
		return FEEDBACK_ACM
	}

	return FEEDBACK_CF
}

type ScoringType int
const (
	SCORING_GROUP ScoringType = iota
	SCORING_SUM
)

func ScoringFromString(str string) ScoringType {
	if str == "group" {
		return SCORING_GROUP
	}

	return SCORING_SUM
}

type Testset struct {
	Name string
	Scoring ScoringType
	Testcases []Testcase
}

func (ts Testset) Score() int {
	switch ts.Scoring {
	case SCORING_GROUP:
		for _, val := range ts.Testcases {
			if val.VerdictName != VERDICT_AC {
				return 0
			}
		}

		return ts.MaxScore()
	case SCORING_SUM:
		sum := 0
		for _, val := range ts.Testcases {
			sum += val.Score
		}

		return sum
	}

	return -1
}

func (ts Testset) MaxScore() int {
	sum := 0
	for _, val := range ts.Testcases {
		sum += val.MaxScore
	}

	return sum
}

func (ts Testset) FirstNonAC() (int) {
	for ind, val := range ts.Testcases {
		if val.VerdictName != VERDICT_AC {
			return ind + 1
		}
	}

	return -1
}

func (ts Testset) IsAC() bool {
	return ts.FirstNonAC() == -1
}

func (ts Testset) MaxMemoryUsage() int {
	mx := 0
	for _, val := range ts.Testcases {
		if mx < val.MemoryUsed {
			mx = val.MemoryUsed
		}
	}

	return mx
}

func (ts Testset) MaxTimeSpent() time.Duration {
	mx := time.Duration(0)
	for _, val := range ts.Testcases {
		if mx < val.TimeSpent {
			mx = val.TimeSpent
		}
	}

	return mx
}

type Testcase struct {
	Testset string
	VerdictName VerdictName
	Score int
	MaxScore int
	Output string
	ExpectedOutput string
	CheckerOutput string
	TimeSpent time.Duration
	MemoryUsed int
}

type Status struct {
	Compiled bool
	FeedbackType FeedbackType
	Feedback []Testset
}

func (v Status) Score() (ans int) {
	for _, f := range v.Feedback {
		ans += f.Score()
	}

	return
}

func (v Status) MaxScore() (ans int) {
	for _, f := range v.Feedback {
		ans += f.MaxScore()
	}

	return
}

func (v Status) Value() (driver.Value, error) {
	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	return buf.String(), nil
}

func (v *Status) Scan(value interface{}) error {
	if value==nil {
		return errors.New("can't scan status from nil")
	}

	if sv, err := driver.String.ConvertValue(value); err == nil {
		var arr []byte

		switch sv.(type) {
		case []uint8:
			orig := sv.([]uint8)
			arr = make([]byte, len(orig))
			for i := 0; i < len(orig); i ++ {
				arr[i] = byte(orig[i])
			}
		case string:
			arr=[]byte(sv.(string))
		}
		rd := bytes.NewReader(arr)

		dec := json.NewDecoder(rd)
		if err := dec.Decode(v); err != nil {
			return err
		}

		return nil
	}

	return errors.New("can't scan from non string type")
}

func (v Status) Verdict() VerdictName {
	if v.FirstNonAC() == -1 {
		return VERDICT_AC
	}

	return v.IndexTestcase(v.FirstNonAC()).VerdictName
}

func (v Status) IsAC() (bool) {
	for _, val := range v.Feedback {
		if !val.IsAC() {
			return false
		}
	}

	return true
}

func (v Status) MaxMemoryUsage() int {
	res := 0
	for _, val := range v.Feedback {
		if res < val.MaxMemoryUsage() {
			res = val.MaxMemoryUsage()
		}
	}

	return res
}

func (v Status) MaxTimeSpent() time.Duration {
	res := time.Duration(0)
	for _, val := range v.Feedback {
		if res < val.MaxTimeSpent() {
			res = val.MaxTimeSpent()
		}
	}

	return res
}

func (v Status) FirstNonAC() (int) {
	ind := 1
	for _, val := range v.Feedback {
		for _, val2 := range val.Testcases {
			if val2.VerdictName != VERDICT_AC {
				return ind
			}

			ind ++
		}
	}

	return -1
}

func (v Status) IndexTestcase(ind int) Testcase {
	curr := 1
	for _, val := range v.Feedback {
		for _, val2 := range val.Testcases {
			if curr == ind {
				return val2
			}

			curr++
		}
	}

	return Testcase{}
}

type Attachment struct {
	Name string
	Contents []byte
}

type Problem interface {
	Name() string

	Titles() []Content
	Statements() []Content
	HTMLStatements() []Content
	PDFStatements() []Content

	MemoryLimit() int
	TimeLimit() int
	InputOutputFiles() (string, string)
	Interactive() bool
	Languages() []language.Language
	Attachments() []Attachment
	Tags() []string

	Compile(language.Sandbox, language.Language, io.Reader, io.Writer) (io.Reader, error)
	Run(language.Sandbox, language.Language, io.Reader, chan string, chan Status) (Status, error)
}