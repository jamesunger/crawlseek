import React from 'react';
import './App.css';

function Navbar() {
    return (
        <nav className="navbar">
            <div className="logo">crawlseek</div>
        </nav>
    );
}

function App() {
    return (
        <div className="App">
            <Navbar />
            <h1>Dungeon Crawl Stone Soup Seed Finder</h1>
            <h2>Choose some constraints below and search for a seed that matches.</h2>
            <form method="post" action="/enqueue">
                <div className="content">
                    {/* Version */}
                    <div className="colContainer">
                        <div className="col">Version</div>
                        <div className="col">
                            <select name="crawl_version">
                                <option value="trunk">trunk</option>
                                <option value="0.24.1">0.24.1</option>
                                <option value="0.25.1">0.25.1</option>
                                <option value="0.26.1">0.26.1</option>
                                <option value="0.28.0">0.28.0</option>
                                <option value="0.29.1">0.29.1</option>
                                <option value="0.30.2">0.30.2</option>
                                <option value="0.31.2">0.31.2</option>
                                <option value="0.32.1">0.32.1</option>
                                <option value="0.33.0">0.33.0</option>
                            </select>
                        </div>
                    </div>

                    {/* Regexp */}
                    <div className="colContainer">
                        <div className="col">Regexp</div>
                        <div className="col">
                            <input type="text" name="regexp" />
                        </div>
                    </div>

                    {/* Attempts */}
                    <div className="colContainer">
                        <div className="col">Attempts</div>
                        <div className="col">
                            <select name="attempts">
                                <option>5</option>
                                <option>10</option>
                                <option>15</option>
                                <option>20</option>
                            </select>
                        </div>
                    </div>

                    {/* Depth */}
                    <div className="colContainer">
                        <div className="col">Depth</div>
                        <div className="col">
                            <select name="depth">
                                <option>5</option>
                                <option>10</option>
                                <option>15</option>
                            </select>
                        </div>
                    </div>

                    {/* Submit Button */}
                    <div className="colContainer">
                        <div className="col">
                            <input type="submit" value="Submit" />
                        </div>
                    </div>
                </div>
            </form>
        </div>
    );
}

export default App;

