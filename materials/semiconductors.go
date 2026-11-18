package materials

// semiconductors carry measured bulk optical constants through the same
// complex-index interface as metals. These are not complete photovoltaic
// coatings, junctions, electrical models or cell-stack prescriptions.
var semiconductors = map[string]page{
	"gallium-arsenide": {parts: map[string]float64{"Ga": 1, "As": 1}, sight: []row{
		{Red, 619.9, 3.878, .211, "gallium arsenide", "main/GaAs/nk/Aspnes.yml", aspnesGaAs, "0.6199 3.878 0.211"},
		{Green, 539.1, 4.100, .320, "gallium arsenide", "main/GaAs/nk/Aspnes.yml", aspnesGaAs, "0.5391 4.100 0.320"},
		{Blue, 459.2, 4.694, .696, "gallium arsenide", "main/GaAs/nk/Aspnes.yml", aspnesGaAs, "0.4592 4.694 0.696"},
	}},
}

// aspnesGaAs identifies the measured sample behind this bulk optical proxy.
const aspnesGaAs = "D. E. Aspnes, S. M. Kelso, R. A. Logan, R. Bhat, Optical properties of AlxGa1-xAs, J. Appl. Phys. 60, 754-767 (1986), doi:10.1063/1.337426"
