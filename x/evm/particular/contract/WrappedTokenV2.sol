// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Permit.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

// WrappedToken is an ERC20 token with minting, burning, voting, and permit functionality, with simple mapping-based minter role management.
contract WrappedTokenV2 is ERC20, ERC20Burnable, ERC20Votes, ERC20Permit, Ownable {
    // Mapping to track minter addresses
    mapping(address => bool) public minters;
    
    // Mapping to track blacklisted addresses
    mapping(address => bool) public blacklists;

    // Event emitted when a minter is added
    event MinterAdded(address indexed minter);
    // Event emitted when a minter is removed
    event MinterRemoved(address indexed minter);
    // Event emitted when an address is added to blacklist
    event BlacklistAdded(address indexed account);
    // Event emitted when an address is removed from blacklist
    event BlacklistRemoved(address indexed account);

    // Modifier to restrict functions to minters
    modifier onlyMinter() {
        require(minters[msg.sender], "caller is not a minter");
        _;
    }

    /**
     * @dev Constructor to initialize the token with name, symbol, initial supply, and initial holder.
     * @param name The name of the token (e.g., "My Token").
     * @param symbol The symbol of the token (e.g., "MTK").
     * @param initialSupply The initial supply of tokens (in wei, accounting for decimals).
     * @param initialHolder The address to receive the initial token supply.
     */
    constructor(
        string memory name,
        string memory symbol,
        uint256 initialSupply,
        address initialHolder
    ) ERC20(name, symbol) ERC20Permit(name) ERC20Votes() Ownable(msg.sender) {
        // Mint the initial supply to the specified holder
        _mint(initialHolder, initialSupply);
    }

    /**
     * @dev Allows the owner to add a minter.
     * @param minter The address to be added as a minter.
     */
    function addMinter(address minter) external onlyOwner {
        require(minter != address(0), "minter cannot be zero address");
        require(!minters[minter], "minter already exists");
        minters[minter] = true;
        emit MinterAdded(minter);
    }

    /**
     * @dev Allows the owner to remove a minter.
     * @param minter The address to be removed from minters.
     */
    function removeMinter(address minter) external onlyOwner {
        require(minters[minter], "minter does not exist");
        minters[minter] = false;
        emit MinterRemoved(minter);
    }

    /**
     * @dev Mints new tokens to the specified address. Only callable by addresses with minter role.
     * @param to The address to receive the newly minted tokens.
     * @param amount The amount of tokens to mint (in wei).
     */
    function mint(address to, uint256 amount) public onlyMinter {
        _mint(to, amount);
    }
    
    /**
     * @dev Adds an address to the blacklist. Only callable by the owner.
     * @param account The address to be added to the blacklist.
     */
    function addBlacklist(address account) external onlyOwner {
        require(account != address(0), "blacklist account is the zero address");
        require(!blacklists[account], "account is already blacklisted");
        blacklists[account] = true;
        emit BlacklistAdded(account);
    }
    
    /**
     * @dev Removes an address from the blacklist. Only callable by the owner.
     * @param account The address to be removed from the blacklist.
     */
    function removeBlacklist(address account) external onlyOwner {
        require(blacklists[account], "account is not blacklisted");
        blacklists[account] = false;
        emit BlacklistRemoved(account);
    }
    
    /**
     * @dev Destroys all tokens from a blacklisted address and reduces total supply.
     * Can only be called by the owner on a blacklisted address.
     * @param account The blacklisted address to destroy tokens from.
     */
    function destroyBlacklistedTokens(address account) external onlyOwner {
        require(blacklists[account], "account is not blacklisted");
        uint256 balance = balanceOf(account);
        require(balance > 0, "account has no balance to destroy");
        
        // Burn the tokens to reduce total supply
        _burn(account, balance);
    }

    /**
     * @dev Overrides the _update function to handle token transfers and voting power updates.
     * @param from The address sending the tokens.
     * @param to The address receiving the tokens.
     * @param value The amount of tokens being transferred (in wei).
     */
    function _update(address from, address to, uint256 value) internal override(ERC20, ERC20Votes) {
        require(!blacklists[from], "account is blacklisted");
        super._update(from, to, value);
    }

    /**
     * @dev Returns the nonce for the given owner, used for permit and voting functionality.
     * Overrides ERC20Permit and Nonces to resolve inheritance conflict.
     * @param owner The address to query the nonce for.
     * @return The current nonce for the owner.
     */
    function nonces(address owner) public view override(ERC20Permit, Nonces) returns (uint256) {
        return super.nonces(owner);
    }
}