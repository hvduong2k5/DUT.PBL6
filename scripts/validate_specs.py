import re
import yaml
import sys

def validate_openapi():
    print("=== Checking OpenAPI 3.0 (docs/03_api_specs/openapi_d2c.yaml) ===")
    with open('docs/03_api_specs/openapi_d2c.yaml', 'r', encoding='utf-8') as f:
        data = yaml.safe_load(f)
    
    paths = data.get('paths', {})
    schemas = data.get('components', {}).get('schemas', {})
    tags = data.get('tags', [])
    
    print(f"Paths count: {len(paths)}")
    print(f"Schemas count: {len(schemas)}")
    print(f"Tags count: {len(tags)}")
    
    # Check all refs
    schema_names = set(schemas.keys())
    broken = []
    
    def check_obj(o, p=''):
        if isinstance(o, dict):
            for k, v in o.items():
                if k == '$ref':
                    if v.startswith('#/components/schemas/'):
                        target = v.split('/')[-1]
                        if target not in schema_names:
                            broken.append((p, v))
                    else:
                        broken.append((p, f"Non-schema ref: {v}"))
                check_obj(v, f"{p}.{k}")
        elif isinstance(o, list):
            for idx, item in enumerate(o):
                check_obj(item, f"{p}[{idx}]")
                
    check_obj(data)
    if broken:
        print(f"FAILED: Found {len(broken)} broken $refs:")
        for b in broken:
            print(f"  {b[0]} -> {b[1]}")
        sys.exit(1)
    else:
        print("PASS: All $refs in OpenAPI are 100% valid!")

def validate_graphql():
    print("\n=== Checking GraphQL Schema (docs/03_api_specs/schema.graphql) ===")
    with open('docs/03_api_specs/schema.graphql', 'r', encoding='utf-8') as f:
        text = f.read()
        
    # Strip docstrings
    text_clean = re.sub(r'"""[\s\S]*?"""', '', text)
    # Strip comments
    text_clean = re.sub(r'#.*', '', text_clean)
    
    # Check brace balance
    open_braces = 0
    for idx, ch in enumerate(text_clean):
        if ch == '{':
            open_braces += 1
        elif ch == '}':
            open_braces -= 1
        if open_braces < 0:
            print(f"FAILED: Unexpected closing brace at character {idx}")
            sys.exit(1)
            
    if open_braces != 0:
        print(f"FAILED: Unbalanced braces count: {open_braces}")
        sys.exit(1)
    print("PASS: Braces balanced perfectly!")
    
    # Check defined types
    builtins = {'String', 'Int', 'Float', 'Boolean', 'ID', 'DateTime', 'JSON'}
    defined = set(builtins)
    for m in re.finditer(r'(?:type|enum|input|scalar)\s+([A-Za-z0-9_]+)', text_clean):
        defined.add(m.group(1))
        
    print(f"Total defined types: {len(defined)}")
    
    # Check references
    missing = set()
    for m in re.finditer(r':\s*\[?([A-Za-z0-9_]+)!?\]?!?', text_clean):
        t = m.group(1)
        if t not in defined:
            missing.add(t)
            
    if missing:
        print(f"FAILED: Missing type definitions: {missing}")
        sys.exit(1)
    else:
        print("PASS: All GraphQL referenced types are defined!")

if __name__ == '__main__':
    validate_openapi()
    validate_graphql()
    print("\n>>> ALL CONTRACT SPECIFICATIONS VALIDATED SUCCESSFULLY! <<<")
