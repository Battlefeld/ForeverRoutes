#!/usr/bin/env python3
"""Package the standalone ForeverRoutes addon with freshly exported route data."""
import argparse
import hashlib
import re
from pathlib import Path
import zipfile


def package(source: Path, routes: Path, output: Path) -> None:
    files = {str(p.relative_to(source)).replace('\\', '/'): p.read_bytes()
             for p in sorted(source.rglob('*')) if p.is_file() and p.suffix in {'.lua', '.toc', '.tga', '.md'}}
    files['Data/Routes.lua'] = routes.read_bytes()
    files['LICENSE.txt'] = Path('LICENSE').read_bytes()
    files['RXP-LICENSE.txt'] = Path('data/fallbacks/RXP-LICENSE.txt').read_bytes()
    toc = files['ForeverRoutes.toc'].decode('utf-8-sig')
    for line in toc.splitlines():
        entry = line.strip().replace('\\', '/')
        if entry and not entry.startswith('#') and entry not in files:
            raise ValueError(f'TOC references missing file: {entry}')
    for media in re.findall(r"file='([^']+)'", files["ArtData.lua"].decode()):
        if "Media/UI/" + media not in files:
            raise ValueError(f"Missing HUD texture: {media}")
    for name, data in files.items():
        if not data:
            raise ValueError(f'Empty addon file: {name}')
    output.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(output, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
        for name, data in sorted(files.items()):
            info = zipfile.ZipInfo('ForeverRoutes/' + name, (2026, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, data)
    output.with_suffix(output.suffix + '.sha256').write_text(
        hashlib.sha256(output.read_bytes()).hexdigest() + '  ' + output.name + '\n')
    print(f'Packaged {len(files)} addon files: {output}')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, default=Path('ForeverRoutes'))
    parser.add_argument('--routes', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    package(args.source, args.routes, args.out)
