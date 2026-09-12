from pathlib import Path
import zipfile,xml.etree.ElementTree as E
p=Path('/tmp/gp-release-review/partial')
with zipfile.ZipFile(p/'control-native.gp') as z: files={n:z.read(n) for n in z.namelist()}
for name,whole,partial,flags,frets in [('below-experiment',5,-3,'001001',[0]*6),('fretted-experiment',0,3,'111111',[0,1,2,3,4,5])]:
 r=E.fromstring(files['Content/score.gpif']);props=r.find('Tracks/Track/Staves/Staff/Properties')
 for key,tag,value in [('CapoFret','Fret',str(whole)),('PartialCapoFret','Fret',str(partial)),('PartialCapoStringFlags','Bitset',flags)]:
  old=props.find(f"Property[@name='{key}']")
  if old is not None:props.remove(old)
  E.SubElement(E.SubElement(props,'Property',{'name':key}),tag).text=value
 for n,v in zip(r.findall('Notes/Note'),frets):n.find("Properties/Property[@name='Fret']/Fret").text=str(v)
 with zipfile.ZipFile(p/(name+'.gp'),'w',zipfile.ZIP_DEFLATED) as z:
  for n,data in files.items():z.writestr(n,E.tostring(r,encoding='utf-8',xml_declaration=True) if n=='Content/score.gpif' else data)
