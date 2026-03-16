package shared

import (
	"io"
	"os"
	"path/filepath"
)

const defaultVersionXML = `<?xml version="1.0" encoding="UTF-8"?>
<hv:version xmlns:hv="http://www.hancom.co.kr/hwpml/2011/version" appVersion="11.0.0.0" hwpxVersion="1.0" />
`

const defaultContainerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="Contents/content.hpf" media-type="application/oebps-package+xml" />
  </rootfiles>
</container>
`

const defaultSettingsXML = `<?xml version="1.0" encoding="UTF-8"?>
<ha:settings xmlns:ha="http://www.hancom.co.kr/hwpml/2011/app">
  <ha:CaretPosition listIDRef="0" paraIDRef="0" pos="0" />
</ha:settings>
`

const defaultContentXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<opf:package
  xmlns:opf="http://www.idpf.org/2007/opf"
  xmlns:hh="http://www.hancom.co.kr/hwpml/2011/head"
  xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"
  xmlns:hs="http://www.hancom.co.kr/hwpml/2011/section"
  version=""
  unique-identifier=""
  id=""
>
  <opf:metadata>
    <opf:title />
    <opf:language>ko</opf:language>
  </opf:metadata>
  <opf:manifest>
    <opf:item id="header" href="Contents/header.xml" media-type="application/xml" />
    <opf:item id="section0" href="Contents/section0.xml" media-type="application/xml" />
    <opf:item id="settings" href="settings.xml" media-type="application/xml" />
  </opf:manifest>
  <opf:spine>
    <opf:itemref idref="header" linear="yes" />
    <opf:itemref idref="section0" linear="yes" />
  </opf:spine>
</opf:package>
`

const defaultHeaderXML = `<?xml version="1.0" encoding="UTF-8"?>
<hh:head
  xmlns:hc="http://www.hancom.co.kr/hwpml/2011/core"
  xmlns:hh="http://www.hancom.co.kr/hwpml/2011/head"
  xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"
  version="1.5"
  secCnt="1"
>
  <hh:beginNum page="1" footnote="1" endnote="1" pic="1" tbl="1" equation="1" />
  <hh:refList>
    <hh:borderFills itemCnt="3">
      <hh:borderFill id="1" threeD="0" shadow="0" centerLine="NONE" breakCellSeparateLine="0">
        <hh:slash type="NONE" Crooked="0" isCounter="0" />
        <hh:backSlash type="NONE" Crooked="0" isCounter="0" />
        <hh:leftBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:rightBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:topBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:bottomBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:diagonal type="SOLID" width="0.1 mm" color="#000000" />
      </hh:borderFill>
      <hh:borderFill id="2" threeD="0" shadow="0" centerLine="NONE" breakCellSeparateLine="0">
        <hh:slash type="NONE" Crooked="0" isCounter="0" />
        <hh:backSlash type="NONE" Crooked="0" isCounter="0" />
        <hh:leftBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:rightBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:topBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:bottomBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:diagonal type="SOLID" width="0.1 mm" color="#000000" />
        <hc:fillBrush>
          <hc:winBrush faceColor="none" hatchColor="#999999" alpha="0" />
        </hc:fillBrush>
      </hh:borderFill>
      <hh:borderFill id="3" threeD="0" shadow="0" centerLine="NONE" breakCellSeparateLine="0">
        <hh:slash type="NONE" Crooked="0" isCounter="0" />
        <hh:backSlash type="NONE" Crooked="0" isCounter="0" />
        <hh:leftBorder type="SOLID" width="0.12 mm" color="#000000" />
        <hh:rightBorder type="SOLID" width="0.12 mm" color="#000000" />
        <hh:topBorder type="SOLID" width="0.12 mm" color="#000000" />
        <hh:bottomBorder type="SOLID" width="0.12 mm" color="#000000" />
        <hh:diagonal type="SOLID" width="0.1 mm" color="#000000" />
      </hh:borderFill>
    </hh:borderFills>
    <hh:binDataList itemCnt="0" />
  </hh:refList>
</hh:head>
`

const defaultSectionXML = `<?xml version="1.0" encoding="UTF-8"?>
<hs:sec
  xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"
  xmlns:hs="http://www.hancom.co.kr/hwpml/2011/section"
>
  <hp:p id="1" paraPrIDRef="0" styleIDRef="0" pageBreak="0" columnBreak="0" merged="0">
    <hp:run charPrIDRef="0">
      <hp:secPr
        id=""
        textDirection="HORIZONTAL"
        spaceColumns="1134"
        tabStop="8000"
        tabStopVal="4000"
        tabStopUnit="HWPUNIT"
        outlineShapeIDRef="1"
        memoShapeIDRef="0"
        textVerticalWidthHead="0"
        masterPageCnt="0"
      >
        <hp:grid lineGrid="0" charGrid="0" wonggojiFormat="0" />
        <hp:startNum pageStartsOn="BOTH" page="0" pic="0" tbl="0" equation="0" />
        <hp:visibility
          hideFirstHeader="0"
          hideFirstFooter="0"
          hideFirstMasterPage="0"
          border="SHOW_ALL"
          fill="SHOW_ALL"
          hideFirstPageNum="0"
          hideFirstEmptyLine="0"
          showLineNumber="0"
        />
        <hp:lineNumberShape restartType="0" countBy="0" distance="0" startNumber="0" />
        <hp:pagePr landscape="WIDELY" width="59528" height="84186" gutterType="LEFT_ONLY">
          <hp:margin header="4252" footer="4252" gutter="0" left="8504" right="8504" top="5668" bottom="4252" />
        </hp:pagePr>
        <hp:footNotePr>
          <hp:autoNumFormat type="DIGIT" userChar="" prefixChar="" suffixChar=")" supscript="0" />
          <hp:noteLine length="-1" type="SOLID" width="0.12 mm" color="#000000" />
          <hp:noteSpacing betweenNotes="283" belowLine="567" aboveLine="850" />
          <hp:numbering type="CONTINUOUS" newNum="1" />
          <hp:placement place="EACH_COLUMN" beneathText="0" />
        </hp:footNotePr>
        <hp:endNotePr>
          <hp:autoNumFormat type="DIGIT" userChar="" prefixChar="" suffixChar=")" supscript="0" />
          <hp:noteLine length="14692344" type="SOLID" width="0.12 mm" color="#000000" />
          <hp:noteSpacing betweenNotes="0" belowLine="567" aboveLine="850" />
          <hp:numbering type="CONTINUOUS" newNum="1" />
          <hp:placement place="END_OF_DOCUMENT" beneathText="0" />
        </hp:endNotePr>
        <hp:pageBorderFill type="BOTH" borderFillIDRef="1" textBorder="PAPER" headerInside="0" footerInside="0" fillArea="PAPER">
          <hp:offset left="1417" right="1417" top="1417" bottom="1417" />
        </hp:pageBorderFill>
        <hp:pageBorderFill type="EVEN" borderFillIDRef="1" textBorder="PAPER" headerInside="0" footerInside="0" fillArea="PAPER">
          <hp:offset left="1417" right="1417" top="1417" bottom="1417" />
        </hp:pageBorderFill>
        <hp:pageBorderFill type="ODD" borderFillIDRef="1" textBorder="PAPER" headerInside="0" footerInside="0" fillArea="PAPER">
          <hp:offset left="1417" right="1417" top="1417" bottom="1417" />
        </hp:pageBorderFill>
      </hp:secPr>
      <hp:ctrl>
        <hp:colPr id="" type="NEWSPAPER" layout="LEFT" colCount="1" sameSz="1" sameGap="0" />
      </hp:ctrl>
    </hp:run>
    <hp:run charPrIDRef="0">
      <hp:t />
    </hp:run>
  </hp:p>
</hs:sec>
`

func copyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Close()
}
