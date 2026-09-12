"""Extract the retained partial-capo fields from native acquisition artifacts."""
import xml.etree.ElementTree as ET
import zipfile

from native_repair_facts import gpif_facts


def partial_capo_facts(path):
    with zipfile.ZipFile(path) as archive:
        root = ET.fromstring(archive.read('Content/score.gpif'))
    staves = []
    for ti, track in enumerate(root.findall('Tracks/Track')):
        for si, staff in enumerate(track.findall('Staves/Staff')):
            props = staff.find('Properties')
            staves.append({
                'track': ti, 'staff': si,
                'whole': int(props.findtext("Property[@name='CapoFret']/Fret", '0')),
                'partial_offset': int(props.findtext("Property[@name='PartialCapoFret']/Fret", '0')),
                'wire_flags': props.findtext("Property[@name='PartialCapoStringFlags']/Bitset"),
                'wire_tuning': [int(v) for v in props.findtext("Property[@name='Tuning']/Pitches", '').split()],
            })
    return {'staves': staves, **gpif_facts(path)}
