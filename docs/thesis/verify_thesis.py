from pathlib import Path
import json
import re

import pdfplumber
from PIL import Image, ImageDraw, ImageFont
from pypdf import PdfReader

ROOT = Path(__file__).resolve().parents[2]
TEMP = ROOT / 'tmp/pdfs/thesis'
PDF = ROOT / 'output/pdf/Property_Co_Pilot_Thesis_EL.pdf'


def contact_sheets():
    page_count = len(PdfReader(PDF).pages)
    pages = sorted(TEMP.glob('page-*.png'))[:page_count]
    font = ImageFont.truetype('C:/Windows/Fonts/arial.ttf', 16)
    for offset in range(0, len(pages), 15):
        sheet = Image.new('RGB', (1100, 1005), '#d5d8dc')
        draw = ImageDraw.Draw(sheet)
        for index, path in enumerate(pages[offset:offset + 15]):
            page = Image.open(path).convert('RGB')
            page.thumbnail((210, 297))
            x = (index % 5) * 220 + 5
            y = (index // 5) * 335 + 5
            sheet.paste(page, (x, y))
            draw.text((x, y + 302), path.stem, fill='black', font=font)
        sheet.save(TEMP / f'contact-{offset // 15 + 1}.png')


def inspect():
    report = json.loads((TEMP / 'layout.json').read_text(encoding='utf-8'))
    reader = PdfReader(PDF)
    issues = []
    pages = []
    with pdfplumber.open(PDF) as pdf:
        for number, page in enumerate(pdf.pages, 1):
            text = page.extract_text() or ''
            body = [char for char in page.chars if 82 < char['top'] < 758]
            outlying = [char for char in body if char['x0'] < 83 or char['x1'] > 512]
            if outlying:
                issues.append({'page': number, 'horizontal_overflow_characters': len(outlying)})
            if '\ufffd' in text or '\x00' in text:
                issues.append({'page': number, 'invalid_character': True})
            if re.search(r'(?i)(postgres(?:ql)?://[^\s]+:[^\s]+@|eyJ[A-Za-z0-9_-]{20,}\.)', text):
                issues.append({'page': number, 'possible_secret': True})
            pages.append({'page': number, 'label': reader.page_labels[number - 1], 'characters': len(text), 'body_characters': len(body)})
    result = {
        'page_count': len(reader.pages),
        'main_body_pages': report['main_body_pages'],
        'issues': issues,
        'sparse_pages': [page for page in pages if page['body_characters'] < 350],
        'pages': pages,
    }
    (TEMP / 'qa.json').write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding='utf-8')
    print(json.dumps({key: value for key, value in result.items() if key != 'pages'}, ensure_ascii=False))
    for entry in report['entries']:
        if entry['kind'] in ('diagram', 'table') or entry['level'] == 0:
            print(entry['physical'], entry['text'])


if __name__ == '__main__':
    contact_sheets()
    inspect()
