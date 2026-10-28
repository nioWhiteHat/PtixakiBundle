from pathlib import Path
import html
import json
import re
import textwrap

from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER, TA_JUSTIFY, TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle
from reportlab.lib.units import cm
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.platypus import BaseDocTemplate, Frame, PageTemplate, Paragraph, Spacer, PageBreak, Table, TableStyle, KeepTogether
from reportlab.graphics.shapes import Drawing, Rect, Line, Polygon, String
from pypdf import PdfReader, PdfWriter

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / 'output/pdf/Property_Co_Pilot_Thesis_EL.pdf'
TEMP = ROOT / 'tmp/pdfs/thesis'
TEMP.mkdir(parents=True, exist_ok=True)
DEST.parent.mkdir(parents=True, exist_ok=True)
FONTDIR = Path('C:/Windows/Fonts')
for name, file in [('Arial', 'arial.ttf'), ('ArialBold', 'arialbd.ttf'), ('ArialItalic', 'ariali.ttf'), ('ArialBlack', 'ariblk.ttf'), ('Code', 'consola.ttf'), ('MathSymbols', 'seguisym.ttf')]:
    pdfmetrics.registerFont(TTFont(name, str(FONTDIR / file)))
pdfmetrics.registerFontFamily('Arial', normal='Arial', bold='ArialBold', italic='ArialItalic', boldItalic='ArialBold')
W, H = A4
M = 3 * cm
WIDTH = W - 2 * M
BLUE = colors.HexColor('#243C50')
LIGHT = colors.HexColor('#F2F4F5')
styles = {
    'body': ParagraphStyle('body', fontName='Arial', fontSize=10, leading=12, alignment=TA_JUSTIFY, spaceAfter=3, firstLineIndent=12),
    'first': ParagraphStyle('first', fontName='Arial', fontSize=10, leading=12, alignment=TA_JUSTIFY, spaceAfter=3),
    'h1': ParagraphStyle('h1', fontName='ArialBlack', fontSize=12, leading=16, spaceBefore=18, spaceAfter=6, keepWithNext=True),
    'h2': ParagraphStyle('h2', fontName='ArialBlack', fontSize=11, leading=14, spaceBefore=18, spaceAfter=6, keepWithNext=True),
    'h3': ParagraphStyle('h3', fontName='ArialBlack', fontSize=10, leading=13, spaceBefore=18, spaceAfter=6, keepWithNext=True),
    'caption': ParagraphStyle('caption', fontName='ArialBold', fontSize=9, leading=11, spaceBefore=4, spaceAfter=6),
    'small': ParagraphStyle('small', fontName='Arial', fontSize=8, leading=10, spaceAfter=4),
    'cell': ParagraphStyle('cell', fontName='Arial', fontSize=9, leading=11, spaceAfter=0),
    'code': ParagraphStyle('code', fontName='Code', fontSize=8, leading=10, leftIndent=5, rightIndent=5, backColor=LIGHT, spaceAfter=0),
    'equation': ParagraphStyle('equation', fontName='Arial', fontSize=10, leading=14, alignment=TA_CENTER, spaceBefore=8, spaceAfter=8),
    'toc': ParagraphStyle('toc', fontName='Arial', fontSize=9, leading=11, spaceAfter=0),
}

def safe(text):
    return html.escape(text.replace('\u2011', '-').replace('–', '-').replace('—', '-'))

def markup(text):
    text = safe(text)
    text = re.sub(r'`([^`]+)`', r'<font name="Code">\1</font>', text)
    text = re.sub(r'\[([^\]]+)\]\((https?://[^)]+)\)', r'<link href="\2" color="#243C50">\1</link>', text)
    for symbol in '₀∈⁻':
        text = text.replace(symbol, '<font name="MathSymbols">'+symbol+'</font>')
    return text

def p(text, style='body'):
    return Paragraph(markup(text), styles[style])

def roman(n):
    result = ''
    for value, symbol in [(1000,'m'), (900,'cm'), (500,'d'), (400,'cd'), (100,'c'), (90,'xc'), (50,'l'), (40,'xl'), (10,'x'), (9,'ix'), (5,'v'), (4,'iv'), (1,'i')]:
        while n >= value:
            result += symbol
            n -= value
    return result

class ThesisDoc(BaseDocTemplate):
    def __init__(self):
        super().__init__(
            str(DEST),
            pagesize=A4,
            leftMargin=M,
            rightMargin=M,
            topMargin=M,
            bottomMargin=M,
            title='Property Co-Pilot',
            author='Αντώνιος Λειβαδίτης',
            allowSplitting=1,
        )
        self.entries = []
        self.main_start = None
        self.back_start = None
        self.addPageTemplates(PageTemplate(id='normal', frames=Frame(M, M, WIDTH, H-2*M, leftPadding=0, rightPadding=0, topPadding=0, bottomPadding=0), onPageEnd=self.footer))

    def footer(self, canvas, doc):
        if doc.page == 1:
            return
        canvas.saveState()
        canvas.setFont('Arial', 8)
        canvas.setFillColor(colors.HexColor('#333333'))
        canvas.drawString(M, H-2.5*cm, 'Πτυχιακή Εργασία')
        canvas.drawRightString(W-M, H-2.5*cm, 'Αντώνιος Λειβαδίτης')
        canvas.drawString(M, 2.5*cm, 'Property Co-Pilot')
        label = str(doc.page-self.main_start+1) if self.main_start else roman(doc.page-1)
        canvas.drawRightString(W-M, 2.5*cm, label)
        canvas.restoreState()

    def afterFlowable(self, flow):
        if getattr(flow, 'start_main', False):
            self.main_start = self.page
        if getattr(flow, 'start_back', False):
            self.back_start = self.page
        if hasattr(flow, 'entry'):
            kind, level, text, key = flow.entry
            label = str(self.page-self.main_start+1) if self.main_start else roman(self.page-1)
            self.entries.append(dict(kind=kind, level=level, text=text, page=label, physical=self.page, key=key))
            self.canv.bookmarkPage(key)
            if kind == 'heading':
                self.canv.addOutlineEntry(text, key, level=min(level, 1), closed=False)

def heading(text, level=0, marker=None):
    f = p(text, 'h'+str(level+1))
    f.entry = ('heading', level, text, 'h_'+str(heading.counter))
    heading.counter += 1
    if marker:
        setattr(f, marker, True)
    return f
heading.counter = 0

def table(rows, widths=None):
    count = len(rows[0])
    if widths is None:
        widths = [WIDTH/count] * count
        if rows[0][0]=='Ακρωνύμιο':
            widths = [58,180,WIDTH-238]
    cells = [[p(str(v), 'cell') for v in row] for row in rows]
    t = Table(cells, colWidths=widths, repeatRows=1, hAlign='LEFT')
    t.setStyle(TableStyle([
        ('BACKGROUND',(0,0),(-1,0),colors.HexColor('#E3E8EB')),
        ('VALIGN',(0,0),(-1,-1),'TOP'),
        ('LINEBELOW',(0,0),(-1,0),0.7,BLUE),
        ('LINEBELOW',(0,1),(-1,-1),0.3,colors.HexColor('#C7CED3')),
        ('LEFTPADDING',(0,0),(-1,-1),5),('RIGHTPADDING',(0,0),(-1,-1),5),
        ('TOPPADDING',(0,0),(-1,-1),5),('BOTTOMPADDING',(0,0),(-1,-1),5),
    ]))
    return t

def diagram(caption, labels):
    def box(index, x, y, width=126):
        height = 40
        d.add(Rect(x-width/2,y-height/2,width,height,rx=3,ry=3,fillColor=LIGHT,strokeColor=BLUE,strokeWidth=.7))
        parts = []
        current = ''
        for word in labels[index].split():
            candidate = (current+' '+word).strip()
            if current and pdfmetrics.stringWidth(candidate,'Arial',8.5)>width-12:
                parts.append(current)
                current = word
            else:
                current = candidate
        parts.append(current)
        for row, part in enumerate(parts):
            d.add(String(x,y+(len(parts)-1)*5.5-row*11-3,part,fontName='Arial',fontSize=8.5,textAnchor='middle',fillColor=BLUE))
        return (x,y,width,height)

    def arrow(points):
        for (x1,y1),(x2,y2) in zip(points,points[1:]):
            d.add(Line(x1,y1,x2,y2,strokeColor=BLUE,strokeWidth=.8))
        x1,y1 = points[-2]
        x2,y2 = points[-1]
        length = ((x2-x1)**2+(y2-y1)**2)**.5
        ux,uy = (x2-x1)/length,(y2-y1)/length
        d.add(Polygon([x2,y2,x2-5*ux-2.5*uy,y2-5*uy+2.5*ux,x2-5*ux+2.5*uy,y2-5*uy-2.5*ux],fillColor=BLUE,strokeColor=None))

    def down(a,b):
        xa,ya,wa,ha = positions[a]
        xb,yb,wb,hb = positions[b]
        start,end = (xa,ya-ha/2),(xb,yb+hb/2)
        mid = (start[1]+end[1])/2
        arrow([start,(xa,mid),(xb,mid),end])

    if 'Επισκόπηση' in caption:
        d = Drawing(WIDTH,270)
        xs = [65,WIDTH/2,WIDTH-65]
        positions = {
            0:box(0,xs[1],245),
            1:box(1,xs[0],175),
            2:box(2,xs[0],105),
            3:box(3,xs[1],175),
            4:box(4,xs[1],105),
            5:box(5,xs[1],35),
            6:box(6,xs[2],175),
            7:box(7,xs[2],105),
        }
        for a,b in [(0,1),(0,3),(0,6),(1,2),(3,4),(4,5),(6,7)]:
            down(a,b)
        x = xs[1]+70
        arrow([(xs[1]+63,175),(x,175),(x,35),(xs[1]+63,35)])
    elif 'Δομή εκτέλεσης' in caption:
        d = Drawing(WIDTH,285)
        xs = [52,159,266,373]
        positions = {
            0:box(0,WIDTH/2,260),
            1:box(1,WIDTH/2,195),
            2:box(2,xs[0],120,98),
            3:box(3,xs[1],120,98),
            4:box(4,xs[2],120,98),
            5:box(5,xs[2],45,98),
            6:box(6,xs[3],120,98),
            7:box(7,xs[3],45,98),
        }
        for a,b in [(0,1),(1,2),(1,3),(1,4),(1,6),(4,5),(6,7)]:
            down(a,b)
    elif 'Λογικός πυρήνας' in caption:
        d = Drawing(WIDTH,290)
        box(0,WIDTH/2,265,180)
        d.add(Line(WIDTH/2,245,WIDTH/2,35,strokeColor=BLUE,strokeWidth=.8))
        for index in range(1,len(labels)):
            row,col = divmod(index-1,2)
            x,y = (95 if col==0 else WIDTH-95),210-row*57
            box(index,x,y,175)
            edge = x+87.5 if col==0 else x-87.5
            d.add(Line(WIDTH/2,y,edge,y,strokeColor=BLUE,strokeWidth=.8))
    else:
        height = ((len(labels)+1)//2)*66
        d = Drawing(WIDTH,height)
        positions = {}
        for index in range(len(labels)):
            row,col = divmod(index,2)
            if row%2:
                col = 1-col
            positions[index] = box(index,WIDTH/4+col*WIDTH/2,height-24-row*66,180)
        for index in range(len(labels)-1):
            xa,ya,wa,ha = positions[index]
            xb,yb,wb,hb = positions[index+1]
            if ya==yb:
                direction = 1 if xb>xa else -1
                arrow([(xa+direction*wa/2,ya),(xb-direction*wb/2,yb)])
            else:
                down(index,index+1)
    return d

def source_block(path, start, end):
    forbidden = ('DbConnection.go','signin.json','cookie.txt','AuthMiddleware.go','SignIn.go','SignUp.go')
    if any(part in path for part in forbidden):
        raise ValueError('Non-public source block: '+path)
    file = ROOT/path
    lines = file.read_text(encoding='utf-8-sig').splitlines()
    end = min(end,len(lines))
    result=[p(f'Πηγή: {path}, γραμμές {start}-{end}. Διατηρείται η αρίθμηση πηγής· παραλείπονται οι γραμμές σχολίων και αναδιπλώνονται οπτικά οι μεγάλες γραμμές.', 'small')]
    for number in range(start,end+1):
        line=lines[number-1].expandtabs(4)
        if line.strip().startswith(('//','#','--')):
            continue
        if not line.strip():
            result.append(Spacer(1,4))
            continue
        chunks=textwrap.wrap(line,width=81,replace_whitespace=False,drop_whitespace=False,break_long_words=True,break_on_hyphens=False) or ['']
        for i,chunk in enumerate(chunks):
            prefix=f'{number:4}  ' if i==0 else '      '
            escaped=safe(prefix+chunk).replace(' ','&#160;')
            result.append(Paragraph(escaped,styles['code']))
    if len(result)>8:
        result = [KeepTogether(result[:4])] + result[4:-3] + [KeepTogether(result[-3:])]
    result.append(Spacer(1,8))
    return result

def parse(text, main=False):
    result=[]
    lines=text.splitlines()
    i=0; chapter=''; table_no=0; diagram_no=0; eq_no=0; first=True
    while i<len(lines):
        line=lines[i].strip()
        if not line:
            i+=1; continue
        if line.startswith('# '):
            if result:
                result.append(PageBreak())
            title=line[2:]
            chapter=title.split('.')[0]
            if title.startswith('Παράρτημα '):
                chapter=title.split('.')[0].split()[-1]
            elif title=='Πίνακας ορολογίας':
                chapter='Ο'
            elif title=='Πίνακας συντμήσεων και ακρωνυμίων':
                chapter='Σ'
            table_no=diagram_no=eq_no=0
            marker='start_main' if main and title.startswith('1.') else ('start_back' if title=='Πίνακας ορολογίας' else None)
            result.append(heading(title,0,marker));first=True;i+=1;continue
        if line.startswith('## '):
            result.append(heading(line[3:],1));first=True;i+=1;continue
        if line.startswith('### '):
            result.append(heading(line[4:],2));first=True;i+=1;continue
        if line.startswith('@source '):
            path,a,b=line[8:].split('|'); result.extend(source_block(path,int(a),int(b)));first=True;i+=1;continue
        if line.startswith('@equation '):
            eq_no+=1
            result.append(p(f'{line[10:]}     ({chapter}.{eq_no})','equation'));i+=1;continue
        if line.startswith('@diagram '):
            data=line[9:].split('|');diagram_no+=1
            title=f'Διάγραμμα {chapter}.{diagram_no}. {data[0]}'
            cap=p(title,'caption');cap.entry=('diagram',0,title,f'd_{chapter}_{diagram_no}')
            result.append(KeepTogether([diagram(data[0],data[1:]),cap]));i+=1;continue
        if line.startswith('@table '):
            table_no+=1;title=f'Πίνακας {chapter}.{table_no}. {line[7:]}'
            cap=p(title,'caption');cap.keepWithNext=True;cap.entry=('table',0,title,f't_{chapter}_{table_no}')
            result.append(cap);i+=1;continue
        if line.startswith('|'):
            rows=[]
            while i<len(lines) and lines[i].strip().startswith('|'):
                row=[cell.strip() for cell in lines[i].strip().strip('|').split('|')]
                if not all(re.fullmatch(r'[-: ]+',cell or '-') for cell in row):rows.append(row)
                i+=1
            result.append(table(rows));result.append(Spacer(1,7));first=True;continue
        if line=='@features':
            manifest=json.loads((ROOT/'Ptixiaki/Worker/Models/TreeOutputs/Phase2_residence_properties_sell_Manifest.json').read_text(encoding='utf-8'))
            rows=[['Θέση','Όνομα χαρακτηριστικού','Encoded όνομα']]+[[str(j),name,'f'+str(j)] for j,name in enumerate(manifest['original_feature_names'])]
            feature_table=table(rows,[44,WIDTH-104,60])
            feature_table.setStyle(TableStyle([('TOPPADDING',(0,0),(-1,-1),4),('BOTTOMPADDING',(0,0),(-1,-1),4)]))
            caption=result.pop()
            result.append(KeepTogether([caption,feature_table]))
            result.append(Spacer(1,7))
            i+=1;continue
        if line in ('@tree','```powershell','```json','```text','```go','```python'):
            stop='@endtree' if line=='@tree' else '```';i+=1
            while i<len(lines) and lines[i].strip()!=stop:
                for chunk in textwrap.wrap(lines[i],width=86,replace_whitespace=False,drop_whitespace=False) or ['']:
                    result.append(Paragraph(safe(chunk).replace(' ','&#160;') or '&#160;',styles['code']))
                i+=1
            result.append(Spacer(1,8));i+=1;first=True;continue
        para=[line];i+=1
        while i<len(lines) and lines[i].strip() and not lines[i].startswith(('#','@','|','```')):
            para.append(lines[i].strip());i+=1
        result.append(p(' '.join(para),'first' if first else 'body'));first=False
    return result

def listing(entries, kind):
    rows=[]
    for item in entries:
        if item['kind']!=kind:continue
        if kind=='heading' and item['text'] in ('Πίνακας περιεχομένων','Κατάλογοι σχημάτων και πινάκων'):continue
        indent='&#160;'*(3*item['level'])
        label=Paragraph(indent+'<link href="#'+item['key']+'">'+safe(item['text'])+'</link>',styles['toc'])
        rows.append([label,p(item['page'],'toc')])
    if not rows:return [p('Ο κατάλογος ενημερώνεται κατά τη στοιχειοθεσία.','small')]
    t=Table(rows,colWidths=[WIDTH-28,28],hAlign='LEFT')
    t.setStyle(TableStyle([('VALIGN',(0,0),(-1,-1),'TOP'),('ALIGN',(-1,0),(-1,-1),'RIGHT'),('LEFTPADDING',(0,0),(-1,-1),0),('RIGHTPADDING',(0,0),(-1,-1),0),('TOPPADDING',(0,0),(-1,-1),2),('BOTTOMPADDING',(0,0),(-1,-1),2)]))
    return [t]

def front(entries):
    title=ParagraphStyle('cover',fontName='ArialBlack',fontSize=22,leading=28,alignment=TA_CENTER,spaceAfter=15)
    center=ParagraphStyle('center',fontName='Arial',fontSize=11,leading=15,alignment=TA_CENTER)
    story=[Spacer(1,25)]
    for text in ['ΠΑΝΕΠΙΣΤΗΜΙΟ ΠΕΙΡΑΙΩΣ','ΣΧΟΛΗ ΤΕΧΝΟΛΟΓΙΩΝ ΠΛΗΡΟΦΟΡΙΚΗΣ','ΚΑΙ ΕΠΙΚΟΙΝΩΝΙΩΝ','ΤΜΗΜΑ ΠΛΗΡΟΦΟΡΙΚΗΣ']:
        story.append(Paragraph(text,center))
    story.extend([Spacer(1,90),Paragraph('Πτυχιακή Εργασία',center),Spacer(1,22),Paragraph('Property Co-Pilot',title),Paragraph('Σύστημα υποστήριξης αποφάσεων για ακίνητα<br/>με μηχανική μάθηση και εξηγήσιμες εκτιμήσεις',center),Spacer(1,15),Paragraph('Property Co-Pilot<br/>A decision-support system for real estate<br/>with machine learning and explainable estimates',center),Spacer(1,65)])
    story.extend([
        p('Ονοματεπώνυμο φοιτητή: Αντώνιος Λειβαδίτης', 'first'),
        p('Αριθμός μητρώου: π22085', 'first'),
        p('Πατρώνυμο: Γεώργιος', 'first'),
        p('Επιβλέπων: Ιωάννης Δαυγιώτης', 'first'),
        Spacer(1, 55),
        Paragraph('Πειραιάς, Οκτώβριος 2026', center),
        PageBreak(),
    ])
    story.append(heading('Πνευματικά δικαιώματα', 0))
    story.extend([
        p('Copyright © 2026. Αντώνιος Λειβαδίτης.', 'first'),
        p('Επιτρέπεται η χρήση της εργασίας για μη εμπορικούς, εκπαιδευτικούς και ερευνητικούς σκοπούς με αναφορά της πηγής. Η εμπορική αναπαραγωγή, αποθήκευση ή διανομή του συνόλου ή μέρους της απαιτεί άδεια του δικαιούχου.'),
        p('Οι απόψεις και τα συμπεράσματα της εργασίας αποδίδονται στον συγγραφέα και δεν αποτελούν επίσημες θέσεις του Πανεπιστημίου Πειραιώς.'),
        PageBreak(),
    ])
    story.append(heading('Περίληψη / Abstract',0))
    story.extend(parse((Path(__file__).parent/'abstract.md').read_text(encoding='utf-8')))
    story.append(PageBreak());story.append(heading('Πίνακας περιεχομένων',0));story.extend(listing(entries,'heading'))
    story.append(PageBreak());story.append(heading('Κατάλογοι σχημάτων και πινάκων',0))
    story.append(p('Κατάλογος διαγραμμάτων','h2'));story.extend(listing(entries,'diagram'))
    story.append(p('Κατάλογος πινάκων','h2'));story.extend(listing(entries,'table'))
    story.append(p('Κατάλογος εικόνων','h2'));story.append(p('Δεν χρησιμοποιούνται φωτογραφίες ή στιγμιότυπα οθόνης. Όλα τα σχήματα είναι πρωτότυπα διανυσματικά διαγράμματα της δομής και των ροών του έργου.','first'))
    story.append(PageBreak())
    return story

def build():
    chapter_files=sorted(Path(__file__).parent.glob('[0-9][0-9]_*.md'))
    entries=[]; previous=None
    for iteration in range(5):
        heading.counter=0
        story=front(entries)
        for number,file in enumerate(chapter_files):
            if number:story.append(PageBreak())
            story.extend(parse(file.read_text(encoding='utf-8'),main=True))
        doc=ThesisDoc();doc.build(story)
        entries=doc.entries
        state=[(x['text'],x['page']) for x in entries]
        if state==previous:break
        previous=state
    report={'physical_pages':doc.page,'main_start':doc.main_start,'back_start':doc.back_start,'main_body_pages':doc.back_start-doc.main_start if doc.back_start else None,'entries':entries}
    writer = PdfWriter(clone_from=PdfReader(DEST))
    writer.set_page_label(0,0,prefix='Εξώφυλλο')
    writer.set_page_label(1,doc.main_start-2,style='/r',start=1)
    writer.set_page_label(doc.main_start-1,doc.page-1,style='/D',start=1)
    labeled = TEMP/'labeled.pdf'
    with labeled.open('wb') as stream:
        writer.write(stream)
    labeled.replace(DEST)
    (TEMP/'layout.json').write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding='utf-8')
    print(json.dumps({k:v for k,v in report.items() if k!='entries'},ensure_ascii=False))
    print(str(DEST))

if __name__=='__main__':
    build()
