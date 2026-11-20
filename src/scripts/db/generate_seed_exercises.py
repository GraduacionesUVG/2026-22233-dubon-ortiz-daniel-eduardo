#!/usr/bin/env python3
"""
Generates scripts/db/seed_exercises.sql from the hasaneyldrm/exercises-dataset
(MIT licensed data, https://github.com/hasaneyldrm/exercises-dataset).

Only a curated list of fundamental gym exercises is exported. Every curated entry
must exist in the source dataset; the script fails loudly otherwise.

Usage:
    python3 scripts/db/generate_seed_exercises.py
"""

import json
import re
import sys
import urllib.request
from pathlib import Path

SOURCE_URL = "https://raw.githubusercontent.com/hasaneyldrm/exercises-dataset/main/data/exercises.json"
OUTPUT_PATH = Path(__file__).resolve().parent / "seed_exercises.sql"

VALID_GROUPS = {"chest", "back", "legs", "shoulders", "arms", "core"}

# Primary mapping: dataset 'target' muscle -> catalog muscle_group.
TARGET_TO_GROUP = {
    "pectorals": "chest",
    "serratus anterior": "chest",
    "lats": "back",
    "upper back": "back",
    "traps": "back",
    "spine": "back",
    "levator scapulae": "back",
    "delts": "shoulders",
    "biceps": "arms",
    "triceps": "arms",
    "forearms": "arms",
    "quads": "legs",
    "glutes": "legs",
    "hamstrings": "legs",
    "calves": "legs",
    "adductors": "legs",
    "abductors": "legs",
    "abs": "core",
}

# Fallback mapping: dataset 'category' (body part) -> catalog muscle_group.
CATEGORY_TO_GROUP = {
    "chest": "chest",
    "back": "back",
    "shoulders": "shoulders",
    "upper arms": "arms",
    "lower arms": "arms",
    "upper legs": "legs",
    "lower legs": "legs",
    "waist": "core",
}

# Curated list: (source name in dataset, name_en, name_es, overrides).
# overrides may contain:
#   "id":    explicit id, used to reuse ids already present in exercise_catalog
#            so ON CONFLICT dedupes them instead of creating duplicates.
#   "group": explicit muscle_group when the dataset target is not mappable.
CURATED = [
    # Chest
    ("barbell bench press", "Barbell Bench Press", "Press de Banca con Barra", {"id": "barbell_bench_press"}),
    ("barbell incline bench press", "Incline Barbell Bench Press", "Press de Banca Inclinado con Barra", {}),
    ("dumbbell bench press", "Dumbbell Bench Press", "Press de Banca con Mancuernas", {}),
    ("dumbbell incline bench press", "Incline Dumbbell Press", "Press Inclinado con Mancuernas", {"id": "dumbbell_incline_press"}),
    ("dumbbell fly", "Dumbbell Fly", "Aperturas con Mancuernas", {}),
    ("cable cross-over variation", "Cable Crossover", "Cruce de Poleas", {}),
    ("lever chest press", "Machine Chest Press", "Press de Pecho en Máquina", {}),
    ("lever seated fly", "Pec Deck Fly", "Aperturas en Máquina (Pec Deck)", {}),
    ("push-up", "Push-Up", "Flexiones de Pecho", {}),
    ("chest dip", "Chest Dip", "Fondos para Pecho", {}),
    ("dumbbell pullover", "Dumbbell Pullover", "Pullover con Mancuerna", {}),
    # Back
    ("pull-up", "Pull Ups", "Dominadas", {"id": "pull_up"}),
    ("chin-up", "Chin-Up", "Dominadas Supinas", {}),
    ("assisted pull-up", "Assisted Pull-Up", "Dominadas Asistidas en Máquina", {}),
    ("cable pulldown", "Lat Pulldown", "Jalón al Pecho", {}),
    ("cable straight arm pulldown", "Straight-Arm Pulldown", "Jalón con Brazos Rectos en Polea", {}),
    ("barbell bent over row", "Barbell Row", "Remo con Barra", {"id": "barbell_row"}),
    ("barbell pendlay row", "Pendlay Row", "Remo Pendlay", {}),
    ("dumbbell one arm bent-over row", "One-Arm Dumbbell Row", "Remo con Mancuerna a Una Mano", {}),
    ("dumbbell incline row", "Chest-Supported Dumbbell Row", "Remo con Mancuernas en Banco Inclinado", {}),
    ("cable seated row", "Seated Cable Row", "Remo Sentado en Polea", {}),
    ("inverted row", "Inverted Row", "Remo Invertido", {}),
    ("hyperextension", "Back Extension", "Hiperextensiones Lumbares", {}),
    ("barbell shrug", "Barbell Shrug", "Encogimientos con Barra", {}),
    # Legs
    ("barbell full squat", "Barbell Back Squat", "Sentadilla con Barra", {"id": "barbell_back_squat"}),
    ("barbell front squat", "Barbell Front Squat", "Sentadilla Frontal con Barra", {}),
    ("dumbbell goblet squat", "Goblet Squat", "Sentadilla Goblet con Mancuerna", {}),
    ("dumbbell single leg split squat", "Bulgarian Split Squat", "Sentadilla Búlgara", {"id": "bulgarian_split_squat"}),
    ("sled hack squat", "Hack Squat", "Sentadilla Hack en Máquina", {}),
    ("sled 45° leg press", "Leg Press", "Prensa de Piernas", {}),
    ("barbell deadlift", "Barbell Deadlift", "Peso Muerto con Barra", {}),
    ("barbell romanian deadlift", "Romanian Deadlift", "Peso Muerto Rumano", {"id": "romanian_deadlift"}),
    ("dumbbell romanian deadlift", "Dumbbell Romanian Deadlift", "Peso Muerto Rumano con Mancuernas", {}),
    ("barbell sumo deadlift", "Sumo Deadlift", "Peso Muerto Sumo", {}),
    ("dumbbell lunge", "Dumbbell Lunge", "Zancadas con Mancuernas", {}),
    ("walking lunge", "Walking Lunge", "Zancadas Caminando", {}),
    ("dumbbell step-up", "Dumbbell Step-Up", "Subidas al Cajón con Mancuernas", {}),
    ("barbell glute bridge", "Barbell Glute Bridge", "Puente de Glúteos con Barra", {}),
    ("lever leg extension", "Leg Extension", "Extensión de Cuádriceps en Máquina", {}),
    ("lever lying leg curl", "Lying Leg Curl", "Curl Femoral Tumbado en Máquina", {}),
    ("lever seated leg curl", "Seated Leg Curl", "Curl Femoral Sentado en Máquina", {}),
    ("barbell standing calf raise", "Standing Barbell Calf Raise", "Elevación de Talones de Pie con Barra", {}),
    ("lever seated calf raise", "Seated Calf Raise", "Elevación de Talones Sentado en Máquina", {}),
    # Shoulders
    ("barbell seated overhead press", "Seated Barbell Overhead Press", "Press Militar Sentado con Barra", {}),
    ("dumbbell seated shoulder press", "Seated Dumbbell Shoulder Press", "Press de Hombros Sentado con Mancuernas", {}),
    ("lever military press", "Machine Shoulder Press", "Press de Hombros en Máquina", {}),
    ("dumbbell arnold press", "Arnold Press", "Press Arnold", {}),
    ("dumbbell lateral raise", "Lateral Raises", "Elevaciones Laterales", {"id": "lateral_raise"}),
    ("cable lateral raise", "Cable Lateral Raise", "Elevación Lateral en Polea", {}),
    ("dumbbell front raise", "Dumbbell Front Raise", "Elevación Frontal con Mancuernas", {}),
    ("dumbbell rear fly", "Dumbbell Rear Delt Fly", "Pájaros con Mancuernas", {}),
    ("cable rear delt row (with rope)", "Cable Rear Delt Row", "Remo para Deltoide Posterior en Polea", {}),
    ("barbell upright row", "Barbell Upright Row", "Remo al Mentón con Barra", {}),
    # Arms
    ("barbell curl", "Barbell Curl", "Curl con Barra", {"id": "barbell_curl"}),
    ("ez barbell curl", "EZ-Bar Curl", "Curl con Barra Z", {}),
    ("dumbbell biceps curl", "Dumbbell Bicep Curl", "Curl de Bíceps con Mancuernas", {}),
    ("dumbbell hammer curl", "Hammer Curl", "Curl Martillo con Mancuernas", {}),
    ("dumbbell incline curl", "Incline Dumbbell Curl", "Curl Inclinado con Mancuernas", {}),
    ("dumbbell concentration curl", "Concentration Curl", "Curl Concentrado con Mancuerna", {}),
    ("barbell preacher curl", "Barbell Preacher Curl", "Curl Predicador con Barra", {}),
    ("cable curl", "Cable Curl", "Curl de Bíceps en Polea", {}),
    ("cable pushdown", "Tricep Pushdown", "Extensión de Tríceps en Polea", {"id": "tricep_pushdown"}),
    ("cable pushdown (with rope attachment)", "Rope Tricep Pushdown", "Extensión de Tríceps en Polea con Cuerda", {}),
    ("cable overhead triceps extension (rope attachment)", "Cable Overhead Triceps Extension", "Extensión de Tríceps sobre la Cabeza en Polea", {}),
    ("barbell lying triceps extension skull crusher", "Skull Crusher", "Press Francés con Barra", {}),
    ("barbell close-grip bench press", "Close-Grip Bench Press", "Press de Banca con Agarre Cerrado", {}),
    ("dumbbell kickback", "Dumbbell Triceps Kickback", "Patada de Tríceps con Mancuerna", {}),
    ("triceps dip", "Triceps Dip", "Fondos para Tríceps", {}),
    ("bench dip (knees bent)", "Bench Dip", "Fondos en Banco", {}),
    # Core
    ("crunch floor", "Crunch", "Abdominales Crunch", {}),
    ("reverse crunch", "Reverse Crunch", "Crunch Inverso", {}),
    ("cable kneeling crunch", "Cable Kneeling Crunch", "Crunch en Polea de Rodillas", {}),
    ("decline sit-up", "Decline Sit-Up", "Abdominales en Banco Declinado", {}),
    ("hanging leg raise", "Hanging Leg Raise", "Elevación de Piernas Colgado", {}),
    ("lying leg raise flat bench", "Lying Leg Raise", "Elevación de Piernas Tumbado", {}),
    ("russian twist", "Russian Twist", "Giros Rusos", {}),
    ("dumbbell side bend", "Dumbbell Side Bend", "Flexión Lateral con Mancuerna", {}),
    ("dead bug", "Dead Bug", "Bicho Muerto (Dead Bug)", {}),
    ("wheel rollerout", "Ab Wheel Rollout", "Rueda Abdominal", {}),
    ("mountain climber", "Mountain Climber", "Escaladores", {"group": "core"}),
    ("lever seated crunch", "Machine Crunch", "Crunch en Máquina", {}),
    ("weighted crunch", "Weighted Crunch", "Crunch con Peso", {}),
    ("cable side bend", "Cable Side Bend", "Flexión Lateral en Polea", {}),
    ("band horizontal pallof press", "Pallof Press", "Press Pallof", {}),
    ("captains chair straight leg raise", "Captain's Chair Leg Raise", "Elevación de Piernas en Silla Romana", {"id": "captains_chair_leg_raise"}),
    ("power point plank", "Plank", "Plancha Abdominal", {}),
    ("side bridge v. 2", "Side Plank", "Plancha Lateral", {}),
    ("cross body crunch", "Cross-Body Crunch", "Crunch Cruzado", {}),
    ("oblique crunches floor", "Oblique Crunch", "Crunch Oblicuo", {}),
    ("jackknife sit-up", "Jackknife Sit-Up", "Abdominales en Navaja", {}),
]


def normalize_name(name: str) -> str:
    # The dataset contains mojibake for the degree sign in some names (e.g. "45в°").
    return name.replace("в°", "°").strip().lower()


def to_snake_case(name: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "_", name.lower())
    return slug.strip("_")


def sql_quote(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def fetch_dataset(url: str) -> list:
    try:
        with urllib.request.urlopen(url, timeout=60) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except Exception as err:
        raise RuntimeError(f"failed to download dataset from {url}: {err}") from err


def resolve_group(source: dict, overrides: dict) -> str:
    if "group" in overrides:
        return overrides["group"]
    group = TARGET_TO_GROUP.get(source.get("target", ""))
    if group is None:
        group = CATEGORY_TO_GROUP.get(source.get("category", ""))
    if group is None:
        raise ValueError(
            f"cannot map '{source['name']}' (target={source.get('target')}, "
            f"category={source.get('category')}) to a muscle group"
        )
    return group


def build_rows(dataset: list) -> list:
    by_name = {}
    for item in dataset:
        by_name.setdefault(normalize_name(item["name"]), item)

    rows, seen_ids, errors = [], set(), []
    for source_name, name_en, name_es, overrides in CURATED:
        source = by_name.get(normalize_name(source_name))
        if source is None:
            errors.append(f"exercise not found in dataset: '{source_name}'")
            continue

        exercise_id = overrides.get("id", to_snake_case(name_en))
        if exercise_id in seen_ids:
            errors.append(f"duplicate id: '{exercise_id}'")
            continue
        seen_ids.add(exercise_id)

        try:
            group = resolve_group(source, overrides)
        except ValueError as err:
            errors.append(str(err))
            continue
        if group not in VALID_GROUPS:
            errors.append(f"invalid muscle group '{group}' for '{source_name}'")
            continue

        for field, value in (("id", exercise_id), ("name_es", name_es), ("name_en", name_en)):
            if len(value) > 100:
                errors.append(f"{field} exceeds 100 chars for '{source_name}'")

        rows.append((exercise_id, name_es, name_en, group))

    if errors:
        raise RuntimeError("curation errors:\n  - " + "\n  - ".join(errors))

    group_order = ["chest", "back", "legs", "shoulders", "arms", "core"]
    rows.sort(key=lambda r: (group_order.index(r[3]), r[0]))
    return rows


def render_sql(rows: list) -> str:
    values = ",\n".join(
        f"({sql_quote(i)}, {sql_quote(es)}, {sql_quote(en)}, {sql_quote(g)})" for i, es, en, g in rows
    )
    return (
        "-- Seed data for exercise_catalog.\n"
        "-- Generated by scripts/db/generate_seed_exercises.py from\n"
        "-- https://github.com/hasaneyldrm/exercises-dataset (data released under the MIT License).\n"
        f"-- Total exercises: {len(rows)}\n\n"
        "INSERT INTO exercise_catalog (id, name_es, name_en, muscle_group) VALUES\n"
        f"{values}\n"
        "ON CONFLICT (id) DO NOTHING;\n"
    )


def main() -> int:
    try:
        dataset = fetch_dataset(SOURCE_URL)
        rows = build_rows(dataset)
    except RuntimeError as err:
        print(f"error: {err}", file=sys.stderr)
        return 1

    OUTPUT_PATH.write_text(render_sql(rows), encoding="utf-8")

    counts = {}
    for row in rows:
        counts[row[3]] = counts.get(row[3], 0) + 1
    print(f"Wrote {len(rows)} exercises to {OUTPUT_PATH}")
    for group, count in counts.items():
        print(f"  {group}: {count}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
