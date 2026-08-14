REM  =====================================================================
REM  UpdateToc.bas - LibreOffice Basic macro
REM  Opens a .docx, updates every text field + every index (TOC),
REM  then re-saves in place as MS Word 2007-365 (.docx).
REM
REM  Invocation (path comes from the DOCX_PATH env var - more reliable
REM  across LO versions than command-line macro arguments):
REM
REM    DOCX_PATH=/abs/path/to/file.docx \
REM    soffice --headless --norestore --nolockcheck \
REM      -env:UserInstallation=file:///abs/path/to/loprofile \
REM      "vnd.sun.star.script:Standard.UpdateToc.Run?language=Basic&location=application"
REM  =====================================================================

Sub Run(Optional sArg As String)
    Dim sPath As String
    Dim sUrl  As String
    Dim oDesk As Object
    Dim oDoc  As Object
    Dim oArgs(1) As New com.sun.star.beans.PropertyValue
    Dim oSaveArgs(0) As New com.sun.star.beans.PropertyValue
    Dim oIndexes As Object
    Dim i As Integer

    If Not IsMissing(sArg) Then
        sPath = sArg
    Else
        sPath = Environ("DOCX_PATH")
    End If
    If sPath = "" Then
        sPath = Environ("DOCX_PATH")
    End If
    If sPath = "" Then
        Stop
    End If

    sUrl = ConvertToURL(sPath)

    oArgs(0).Name  = "Hidden"      : oArgs(0).Value = True
    oArgs(1).Name  = "UpdateDocMode"
    oArgs(1).Value = com.sun.star.document.UpdateDocMode.FULL_UPDATE

    oDesk = createUnoService("com.sun.star.frame.Desktop")
    oDoc  = oDesk.loadComponentFromURL(sUrl, "_blank", 0, oArgs())

    ' 1. Refresh all text fields (PAGE, NUMPAGES, DATE, ...)
    On Error Resume Next
    oDoc.getTextFields().refresh()
    On Error Goto 0

    ' 2. Rebuild every document index / TOC.  Do it twice: the first pass
    '    inserts the entries (which reflows the document and therefore
    '    changes page numbers), the second pass fixes the page numbers.
    oIndexes = oDoc.getDocumentIndexes()
    For i = 0 To oIndexes.getCount() - 1
        oIndexes.getByIndex(i).update()
    Next i

    ' force a full layout pass before the second update
    On Error Resume Next
    oDoc.getTextFields().refresh()
    oDoc.refresh()
    On Error Goto 0

    For i = 0 To oIndexes.getCount() - 1
        oIndexes.getByIndex(i).update()
    Next i

    On Error Resume Next
    oDoc.refresh()
    On Error Goto 0

    ' 3. Save back as .docx in place
    oSaveArgs(0).Name  = "FilterName"
    oSaveArgs(0).Value = "MS Word 2007 XML"
    oDoc.storeToURL(sUrl, oSaveArgs())

    oDoc.close(False)
End Sub

Sub ToPdf(Optional sArg As String)
    Dim sPath As String, sOut As String, sUrl As String
    Dim oDesk As Object, oDoc As Object
    Dim oArgs(1) As New com.sun.star.beans.PropertyValue
    Dim oPdf(0) As New com.sun.star.beans.PropertyValue
    Dim oIndexes As Object
    Dim i As Integer

    sPath = Environ("DOCX_PATH")
    sOut  = Environ("PDF_PATH")
    sUrl  = ConvertToURL(sPath)

    oArgs(0).Name = "Hidden" : oArgs(0).Value = True
    oArgs(1).Name = "UpdateDocMode"
    oArgs(1).Value = com.sun.star.document.UpdateDocMode.FULL_UPDATE

    oDesk = createUnoService("com.sun.star.frame.Desktop")
    oDoc  = oDesk.loadComponentFromURL(sUrl, "_blank", 0, oArgs())

    On Error Resume Next
    oDoc.getTextFields().refresh()
    On Error Goto 0
    oIndexes = oDoc.getDocumentIndexes()
    For i = 0 To oIndexes.getCount() - 1
        oIndexes.getByIndex(i).update()
    Next i
    On Error Resume Next
    oDoc.refresh()
    On Error Goto 0
    For i = 0 To oIndexes.getCount() - 1
        oIndexes.getByIndex(i).update()
    Next i

    oPdf(0).Name = "FilterName" : oPdf(0).Value = "writer_pdf_Export"
    oDoc.storeToURL(ConvertToURL(sOut), oPdf())
    oDoc.close(False)
End Sub
