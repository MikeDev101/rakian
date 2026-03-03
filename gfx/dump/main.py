from pathlib import Path
from PIL import Image

def convert_pbm_to_bmp_recursively(source_dir, dest_dir):
    source_path = Path(source_dir)
    dest_path = Path(dest_dir)

    if not source_path.is_dir():
        print(f"Error: Source directory '{source_dir}' not found.")
        return

    pbm_files = list(source_path.rglob("*.pbm"))
    
    if not pbm_files:
        print(f"No .pbm files found in '{source_dir}'.")
        return

    print(f"Found {len(pbm_files)} .pbm files. Starting conversion...\n")

    success_count = 0
    error_count = 0

    for pbm_file in pbm_files:
        relative_path = pbm_file.relative_to(source_path)
        target_file = dest_path / relative_path.with_suffix('.bmp')
        target_file.parent.mkdir(parents=True, exist_ok=True)
        
        try:
            with Image.open(pbm_file) as img:
                img.save(target_file)
            print(f"Converted: {relative_path.name}")
            success_count += 1
        except Exception as e:
            print(f"Failed to convert {relative_path.name}: {e}")
            error_count += 1

    # Print a summary
    print("\n--- Conversion Complete ---")
    print(f"Successfully converted: {success_count}")
    if error_count > 0:
        print(f"Failed to convert: {error_count}")

convert_pbm_to_bmp_recursively("./3310", "./3310-converted")
convert_pbm_to_bmp_recursively("./5110", "./5110-converted")