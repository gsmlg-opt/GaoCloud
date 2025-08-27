package g53

import (
	"bytes"
	"testing"
)

func TestDigitualNameParse(t *testing.T) {
	digitalName := NameFromStringUnsafe("ab.\\208L/\003\\248\236/\003")
	Assert(t, bytes.Compare(digitalName.raw, []byte{2, 97, 98, 8, 208, 108, 47, 3, 248, 158, 47, 3, 0}) == 0, "parse string failed")

	desiredString := "ab.\\208l/\\003\\248\\158/\\003"
	Assert(t, digitalName.String(true) == desiredString, "digital name parse then to string failed")
}

func TestNameConcat(t *testing.T) {
	name, _ := Root.Concat(
		NameFromStringUnsafe("a"),
		NameFromStringUnsafe("b"),
		NameFromStringUnsafe("c"),
		NameFromStringUnsafe("d"),
		NameFromStringUnsafe("e"),
		NameFromStringUnsafe("f"),
		NameFromStringUnsafe("g"),
		NameFromStringUnsafe("h"),
	)
	NameEqToStr(t, name, "a.b.c.d.e.f.g.h")

	name, _ = NameFromStringUnsafe("a").Concat(
		NameFromStringUnsafe("b"),
		NameFromStringUnsafe("c"),
		NameFromStringUnsafe("d"),
		NameFromStringUnsafe("e"),
		NameFromStringUnsafe("f"),
		NameFromStringUnsafe("g"),
		NameFromStringUnsafe("h"),
	)
	NameEqToStr(t, name, "a.b.c.d.e.f.g.h")

	name, _ = NameFromStringUnsafe("a").Concat(Root)
	NameEqToStr(t, name, "a")
}

func TestNameSplit(t *testing.T) {
	wwwknetcn, _ := NewName("www.knet.Cn", true)
	n, _ := wwwknetcn.Split(0, 1)
	NameEqToStr(t, n, "www")

	n, _ = wwwknetcn.Split(0, 4)
	NameEqToStr(t, n, "www.knet.cn")

	n, _ = wwwknetcn.Split(1, 3)
	NameEqToStr(t, n, "knet.cn")

	n, _ = wwwknetcn.Split(1, 2)
	NameEqToStr(t, n, "knet.cn")

	n, _ = wwwknetcn.Parent(0)
	NameEqToStr(t, n, "www.knet.cn")

	n, _ = wwwknetcn.Parent(1)
	NameEqToStr(t, n, "knet.cn")

	n, _ = wwwknetcn.Parent(2)
	NameEqToStr(t, n, "cn")

	n, _ = wwwknetcn.Parent(3)
	NameEqToStr(t, n, ".")

	_, err := wwwknetcn.Parent(4)
	Assert(t, err != nil, "www.knet.cn has no parent leve 4")

	n, _ = wwwknetcn.Subtract(Root)
	Assert(t, n.Equals(wwwknetcn), "name substract root equals itself")
}

func TestNameCompare(t *testing.T) {
	knetmixcase, _ := NewName("www.KNET.cN", false)
	knetdowncase, _ := NewName("www.knet.cn", true)
	knetmixcase.Downcase()
	cr := knetmixcase.Compare(knetdowncase, true)
	Assert(t, cr.Order == 0 && cr.CommonLabelCount == 4 && cr.Relation == EQUAL, "down case failed:%v")

	baidu_com, _ := NewName("baidu.com.", true)
	www_baidu_com, _ := NewName("www.baidu.com", true)
	cr = baidu_com.Compare(www_baidu_com, true)
	Assert(t, cr.Relation == SUPERDOMAIN, "baidu.com is www.baidu.com's superdomain but get %v", cr.Relation)

	baidu_cn, _ := NewName("baidu.cn.", true)
	cr = baidu_com.Compare(baidu_cn, true)
	Assert(t, cr.Relation == COMMONANCESTOR && cr.CommonLabelCount == 1, "baidu.com don't have any relationship with baidu.cn")
}

func TestNameReverse(t *testing.T) {
	knetcn, _ := NewName("www.knet.Cn", true)
	knetcnReverse := knetcn.Reverse().String(false)
	Assert(t, knetcnReverse == "cn.knet.www.", "www.knet.com reverse should be com.baidu.www. but get %v", knetcnReverse)

	Assert(t, Root.Reverse().String(false) == ".", "rootcom reverse should be .")
}

func TestNameStrip(t *testing.T) {
	knetmixcase, _ := NewName("www.KNET.cN", false)
	knetWithoutCN, _ := knetmixcase.StripLeft(1)
	NameEqToStr(t, knetWithoutCN, "knet.cn")

	cn, _ := knetmixcase.StripLeft(2)
	NameEqToStr(t, cn, "cn")

	root, _ := knetmixcase.StripLeft(3)
	NameEqToStr(t, root, ".")

	knettld, _ := knetmixcase.StripRight(1)
	NameEqToStr(t, knettld, "www.knet")

	wwwtld, _ := knetmixcase.StripRight(2)
	NameEqToStr(t, wwwtld, "www")

	Equal(t, wwwtld.String(true), "www")
	Equal(t, wwwtld.String(false), "www.")

	root, _ = knetmixcase.StripRight(3)
	NameEqToStr(t, root, ".")
}

func TestNameHash(t *testing.T) {
	name1, _ := NewName("wwwnnnnnnnnnnnnn.KNET.cNNNNNNNNN", false)
	name2, _ := NewName("wwwnnnnnnnnnnnnn.KNET.cNNNNNNNNn", false)
	name3, _ := NewName("wwwnnnnnnnnnnnnn.KNET.cNNNNNNNNN.baidu.com.cn.net", false)
	Equal(t, name1.Hash(false), name2.Hash(false))
	Assert(t, name1.Hash(false) != name3.Hash(false), "different name should has different hash")

	name1, _ = NewName("a.example.com", false)
	name2, _ = NewName("b.example.com", false)
	Assert(t, name1.Hash(true) != name2.Hash(true), "different name should has different hash")

	//name collision
	//Assert(t, NameFromStringUnsafe("2298.com").Hash(false) != NameFromStringUnsafe("23yy.com").Hash(false), "")
}

func TestNameIsSubdomain(t *testing.T) {
	www_knet_cn, _ := NewName("www.knet.Cn", true)
	www_knet, _ := NewName("www.knet", true)
	knet_cn, _ := NewName("knet.Cn", false)
	cn, _ := NewName("cn", true)
	knet, _ := NewName("kNeT", false)

	Assert(t, www_knet_cn.IsSubDomain(knet_cn) &&
		knet_cn.IsSubDomain(cn) &&
		www_knet.IsSubDomain(knet) &&
		knet_cn.IsSubDomain(Root) &&
		cn.IsSubDomain(Root) &&
		knet.IsSubDomain(Root) &&
		www_knet_cn.IsSubDomain(Root) &&
		www_knet.IsSubDomain(Root) &&
		Root.IsSubDomain(Root), "sub domain test fail")

	Assert(t, knet.IsSubDomain(knet_cn) == false &&
		knet.IsSubDomain(cn) == false &&
		Root.IsSubDomain(cn) == false &&
		www_knet.IsSubDomain(www_knet_cn) == false, "kent isnot sub domain of knet.cn or cn")
}

func TestNameEquals(t *testing.T) {
	knetmixcase, _ := NewName("www.KNET.cN", false)
	knetdowncase, _ := NewName("www.knet.cn", true)
	Assert(t, knetmixcase.Equals(knetdowncase), "www.knet.cn is same with www.KNET.cN")

	Assert(t, knetmixcase.CaseSensitiveEquals(knetdowncase) == false, "www.knet.cn isnot casesenstive same with www.KNET.cN")

	knetmixcase.Downcase()
	Assert(t, knetmixcase.CaseSensitiveEquals(knetdowncase), "")
}
