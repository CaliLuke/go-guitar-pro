"""Read native Guitar Pro evidence without invoking or simulating the application."""
import xml.etree.ElementTree as ET
import zipfile
from pathlib import Path


def gpif_facts(path):
    with zipfile.ZipFile(path) as archive:
        root = ET.fromstring(archive.read('Content/score.gpif'))
    lookup = {kind: {item.get('id'): item for item in root.findall(f'{kind}s/{kind}')}
              for kind in ['Bar', 'Voice', 'Beat', 'Note', 'Rhythm']}
    occurrences = []
    for mi, master in enumerate(root.findall('MasterBars/MasterBar')):
        for si, bar_id in enumerate(master.findtext('Bars', '').split()):
            bar = lookup['Bar'][bar_id]
            for vi, voice_id in enumerate(bar.findtext('Voices', '').split()):
                if voice_id == '-1':
                    continue
                for bi, beat_id in enumerate(lookup['Voice'][voice_id].findtext('Beats', '').split()):
                    beat = lookup['Beat'][beat_id]
                    rhythm_ref = beat.find('Rhythm')
                    rhythm = lookup['Rhythm'].get(rhythm_ref.get('ref')) if rhythm_ref is not None else None
                    for ni, note_id in enumerate(beat.findtext('Notes', '').split()):
                        properties = {}
                        for prop in lookup['Note'][note_id].findall('Properties/Property'):
                            name = prop.get('name')
                            if name in ['Fret', 'Midi', 'String', 'HarmonicFret', 'HarmonicType']:
                                properties[name] = ''.join(prop.itertext()).strip()
                            elif name in ['ConcertPitch', 'TransposedPitch']:
                                pitch = prop.find('Pitch')
                                properties[name] = {key: pitch.findtext(key, '') for key in ['Step', 'Accidental', 'Octave']}
                        occurrences.append({'path': [mi, si, vi, bi, ni], 'properties': properties,
                                            'grace': beat.findtext('GraceNotes'),
                                            'rhythm': [(child.tag, child.text, child.attrib) for child in rhythm] if rhythm is not None else None})
    events = [{key: event.findtext(key) for key in ['Type', 'Bar', 'Position', 'Value', 'Linear']}
              for event in root.findall('.//Automation')]
    return {'note_occurrences': occurrences, 'automations': events}


def midi_facts(path):
    data = Path(path).read_bytes()
    if data[:4] != b'MThd' or int.from_bytes(data[4:8], 'big') != 6:
        raise ValueError('unsupported MIDI header')
    division = int.from_bytes(data[12:14], 'big')
    position = 14
    tracks = []
    while position < len(data):
        if data[position:position+4] != b'MTrk':
            raise ValueError('invalid MIDI track')
        length = int.from_bytes(data[position+4:position+8], 'big')
        track = data[position+8:position+8+length]
        position += 8 + length
        index = tick = status = 0
        events = []

        def vlq():
            nonlocal index
            value = 0
            while True:
                byte = track[index]
                index += 1
                value = (value << 7) | (byte & 127)
                if byte < 128:
                    return value

        while index < len(track):
            tick += vlq()
            if track[index] >= 128:
                status = track[index]
                index += 1
            if status == 255:
                index += 1
                count = vlq()
                index += count
                continue
            if status in (240, 247):
                count = vlq()
                index += count
                continue
            count = 1 if status >> 4 in (12, 13) else 2
            values = list(track[index:index+count])
            index += count
            if status >> 4 in (8, 9):
                events.append([tick, status, *values])
        tracks.append(events)
    return {'division': division, 'note_events': tracks}
