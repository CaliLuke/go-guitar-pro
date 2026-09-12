from pathlib import Path
import zipfile,xml.etree.ElementTree as ET
folder=Path('/tmp/gp-release-review/partial')
with zipfile.ZipFile(folder/'control-input.gp') as z: files={x:z.read(x) for x in z.namelist()}
for name,flags in [('experiment-first','100000'),('experiment-fourth','000100'),('experiment-asymmetric','100100')]:
 root=ET.fromstring(files['Content/score.gpif'])
 props=root.find('Tracks/Track/Staves/Staff/Properties')
 for key,tag,value in [('PartialCapoFret','Fret','3'),('PartialCapoStringFlags','Bitset',flags)]:
  old=props.find(f"Property[@name='{key}']")
  if old is not None:props.remove(old)
  p=ET.SubElement(props,'Property',{'name':key});ET.SubElement(p,tag).text=value
 with zipfile.ZipFile(folder/(name+'.gp'),'w',zipfile.ZIP_DEFLATED) as z:
  for path,data in files.items():z.writestr(path,ET.tostring(root,encoding='utf-8',xml_declaration=True) if path=='Content/score.gpif' else data)
