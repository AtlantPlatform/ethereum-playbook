// Copyright 2017, 2018 Tensigma Ltd. All rights reserved.
// Use of this source code is governed by Microsoft Reference Source
// License (MS-RSL) that can be found in the LICENSE file.

pragma solidity ^0.4.24;

import "../BaseSecurityToken/contracts/BaseSecurityToken.sol";

contract PropertyToken is BaseSecurityToken, Owned {

  uint public limit = 50 * 1e6 * 1e18;
  string public name = "Atlant Property Token 000";
  string public symbol = "PTO000";
  uint8 public decimals = 18;

  constructor(string _name, string _symbol, uint _limit) public {
    require(_limit != 0);

    name = _name;
    symbol = _symbol;
    limit = _limit;
  }

  function mint(address _holder, uint _value) external onlyOwner {
    require(_value != 0);
    require(totalSupply + _value <= limit);

    balances[_holder] += _value;
    totalSupply += _value;

    emit Transfer(0x0, _holder, _value);
  }
}
