#!/usr/bin/env python3

import os
import re
from pathlib import Path

# Map of existing themes (normalized names to check for duplicates)
EXISTING_THEMES = {
    "vscodedark", "vscodelight", "hybrid", "hybridlight",
    "rosepine", "rosepinedawn", "ubuntu", "ubuntulight",
    "dracula", "draculalight", "githubdark", "githublight",
    "gruvboxdark", "gruvboxlight", "solarizeddark", "solarizedlight",
    "tokyonight", "tokyonightlight"
}

def parse_ghostty_theme(file_path):
    """Parse a Ghostty theme file and return theme data."""
    theme = {}
    
    with open(file_path, 'r') as f:
        content = f.read()
    
    # Parse palette colors (0-15)
    palette = {}
    palette_matches = re.findall(r'palette\s*=\s*(\d+)=#([0-9a-fA-F]{6})', content)
    for index, color in palette_matches:
        palette[int(index)] = f"#{color}"
    
    # Parse other colors
    background_match = re.search(r'background\s*=\s*#([0-9a-fA-F]{6})', content)
    foreground_match = re.search(r'foreground\s*=\s*#([0-9a-fA-F]{6})', content)
    cursor_match = re.search(r'cursor-color\s*=\s*#([0-9a-fA-F]{6})', content)
    
    # Map to xterm.js theme format
    theme_data = {
        'foreground': f"#{foreground_match.group(1)}" if foreground_match else "#ffffff",
        'background': f"#{background_match.group(1)}" if background_match else "#000000",
    }
    
    if cursor_match:
        theme_data['cursor'] = f"#{cursor_match.group(1)}"
    
    # Map palette indices to terminal colors
    color_mapping = {
        0: 'black',
        1: 'red',
        2: 'green',
        3: 'yellow',
        4: 'blue',
        5: 'magenta',
        6: 'cyan',
        7: 'white',
        8: 'brightBlack',
        9: 'brightRed',
        10: 'brightGreen',
        11: 'brightYellow',
        12: 'brightBlue',
        13: 'brightMagenta',
        14: 'brightCyan',
        15: 'brightWhite'
    }
    
    for idx, color_name in color_mapping.items():
        if idx in palette:
            theme_data[color_name] = palette[idx]
    
    return theme_data

def format_theme_ts(theme_name, theme_data):
    """Format theme data as TypeScript code."""
    var_name = re.sub(r'[^a-zA-Z0-9]', '', theme_name)
    # Ensure variable name doesn't start with a number
    if var_name and var_name[0].isdigit():
        var_name = 'theme' + var_name
    var_name = var_name[0].lower() + var_name[1:] if var_name else 'theme'
    
    lines = [f"const {var_name}: ITheme = {{"]
    
    # Add properties in a specific order
    props_order = ['foreground', 'background', 'cursor', 'black', 'red', 'green', 'yellow', 
                   'blue', 'magenta', 'cyan', 'white', 'brightBlack', 'brightRed', 
                   'brightGreen', 'brightYellow', 'brightBlue', 'brightMagenta', 
                   'brightCyan', 'brightWhite']
    
    for prop in props_order:
        if prop in theme_data:
            lines.append(f'  {prop}: "{theme_data[prop]}",')
    
    lines.append("};")
    lines.append("")
    
    return '\n'.join(lines), var_name

def main():
    ghostty_dir = Path("/home/ovidiu/Workspace/iTerm2-Color-Schemes/ghostty")
    
    themes_to_add = []
    theme_vars = []
    
    # Process all Ghostty theme files
    for theme_file in sorted(ghostty_dir.iterdir()):
        if theme_file.is_file():
            theme_name = theme_file.name
            
            # Check if theme already exists (normalize name for comparison)
            normalized_name = re.sub(r'[^a-z0-9]', '', theme_name.lower())
            
            if normalized_name in EXISTING_THEMES:
                print(f"Skipping existing theme: {theme_name}")
                continue
            
            try:
                theme_data = parse_ghostty_theme(theme_file)
                theme_ts, var_name = format_theme_ts(theme_name, theme_data)
                
                themes_to_add.append((theme_name, theme_ts, var_name))
                print(f"Converted: {theme_name}")
            except Exception as e:
                print(f"Error converting {theme_name}: {e}")
    
    # Generate the TypeScript file content
    output = []
    output.append('import type { ITheme } from "sshx-xterm";')
    output.append('')
    output.append('// Existing themes')
    output.append('')
    
    # Add placeholder for existing themes (we'll append to the existing file)
    output.append('// ... existing theme definitions ...')
    output.append('')
    output.append('// New themes imported from Ghostty')
    output.append('')
    
    # Add all new theme definitions
    for theme_name, theme_ts, var_name in themes_to_add:
        output.append(f"// {theme_name}")
        output.append(theme_ts)
        theme_vars.append((theme_name, var_name))
    
    # Generate the themes object
    output.append("// Add these to the themes object:")
    output.append("const newThemes = {")
    for theme_name, var_name in theme_vars:
        # Clean up theme name for display
        display_name = theme_name.replace('-', ' ').replace('_', ' ')
        output.append(f'  "{display_name}": {var_name},')
    output.append("};")
    
    # Write to a temporary file for review
    output_file = Path("/home/ovidiu/Workspace/SSHXtend/new_themes.ts")
    with open(output_file, 'w') as f:
        f.write('\n'.join(output))
    
    print(f"\nGenerated {len(themes_to_add)} new themes")
    print(f"Output written to: {output_file}")
    
    # Also generate just the theme definitions to append
    definitions_only = []
    for theme_name, theme_ts, var_name in themes_to_add:
        definitions_only.append(f"// {theme_name}")
        definitions_only.append(theme_ts)
    
    definitions_file = Path("/home/ovidiu/Workspace/SSHXtend/theme_definitions.ts")
    with open(definitions_file, 'w') as f:
        f.write('\n'.join(definitions_only))
    
    # Generate the themes object additions
    additions = []
    for theme_name, var_name in theme_vars:
        display_name = theme_name.replace('-', ' ').replace('_', ' ')
        additions.append(f'  "{display_name}": {var_name},')
    
    additions_file = Path("/home/ovidiu/Workspace/SSHXtend/theme_additions.ts")
    with open(additions_file, 'w') as f:
        f.write('\n'.join(additions))

if __name__ == "__main__":
    main()