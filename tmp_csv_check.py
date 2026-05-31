import csv
path = r'c:\Users\lenovo\Desktop\校友分会分类.csv'
with open(path, encoding='utf-8', errors='replace') as f:
    reader = csv.reader(f)
    rows = list(reader)
    print('rows', len(rows))
    for i, row in enumerate(rows[:20]):
        print(i, row)
