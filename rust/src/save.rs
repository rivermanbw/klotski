use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

/// PlayerData holds one player's league progress.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlayerData {
    pub scores: HashMap<usize, i32>, // puzzle index -> best score
}

impl PlayerData {
    pub fn new() -> Self {
        Self {
            scores: HashMap::new(),
        }
    }

    /// Sum of best scores across all completed puzzles.
    pub fn total_score(&self) -> i32 {
        self.scores.values().sum()
    }

    /// Number of puzzles with a score.
    pub fn completed(&self) -> usize {
        self.scores.len()
    }
}

/// SaveData is the top-level persistent state written to disk.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SaveData {
    pub last_player: String,
    pub players: HashMap<String, PlayerData>,
}

impl SaveData {
    pub fn new() -> Self {
        Self {
            last_player: String::new(),
            players: HashMap::new(),
        }
    }

    pub fn player(&mut self, name: &str) -> &mut PlayerData {
        self.players
            .entry(name.to_string())
            .or_insert_with(PlayerData::new)
    }

    pub fn player_ref(&self, name: &str) -> Option<&PlayerData> {
        self.players.get(name)
    }

    pub fn save(&self) -> Result<(), Box<dyn std::error::Error>> {
        let dir = save_dir();
        fs::create_dir_all(&dir)?;
        let data = serde_json::to_string_pretty(self)?;
        fs::write(save_path(), data)?;
        Ok(())
    }

    pub fn leaderboard(&self) -> Vec<LeaderboardEntry> {
        let mut entries: Vec<LeaderboardEntry> = self
            .players
            .iter()
            .map(|(name, pd)| LeaderboardEntry {
                name: name.clone(),
                total: pd.total_score(),
                completed: pd.completed(),
            })
            .collect();
        entries.sort_by(|a, b| b.total.cmp(&a.total).then(a.name.cmp(&b.name)));
        entries
    }
}

#[derive(Debug, Clone)]
pub struct LeaderboardEntry {
    pub name: String,
    pub total: i32,
    pub completed: usize,
}

fn save_dir() -> PathBuf {
    let home = dirs_or_home();
    home.join(".klotski-puzzle")
}

fn save_path() -> PathBuf {
    save_dir().join("save.json")
}

fn dirs_or_home() -> PathBuf {
    if let Some(home) = std::env::var_os("HOME") {
        PathBuf::from(home)
    } else {
        PathBuf::from(".")
    }
}

pub fn load_save() -> SaveData {
    let path = save_path();
    match fs::read_to_string(&path) {
        Ok(data) => match serde_json::from_str::<SaveData>(&data) {
            Ok(s) => s,
            Err(_) => SaveData::new(),
        },
        Err(_) => SaveData::new(),
    }
}
